package main

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// RemoteTarget is a parsed "user@host:port:/path" SSH target.
// IPv6 hosts are not supported yet.
type RemoteTarget struct {
	User string
	Host string
	Port int
	Path string
}

// ParseRemoteTarget parses "user@host:port:/path" or "host:path".
func ParseRemoteTarget(s, defaultUser string, defaultPort int) (*RemoteTarget, error) {
	if defaultPort == 0 {
		defaultPort = 22
	}
	rest := s
	uname := defaultUser
	if at := strings.LastIndex(s, "@"); at >= 0 {
		uname = s[:at]
		rest = s[at+1:]
	}
	// split on the LAST colon: everything before is host[:port], after is path
	// (paths can contain colons, host:port syntax cannot be ambiguous)
	colon := strings.LastIndex(rest, ":")
	if colon < 0 {
		return nil, fmt.Errorf("target %q: want user@host:/path/to/.env", s)
	}
	hostPort := rest[:colon]
	path := rest[colon+1:]
	if hostPort == "" || path == "" {
		return nil, fmt.Errorf("target %q: empty host or path", s)
	}
	host := hostPort
	port := defaultPort
	if hp := strings.Split(hostPort, ":"); len(hp) == 2 {
		host = hp[0]
		p, err := strconv.Atoi(hp[1])
		if err != nil || p <= 0 || p >= 65536 {
			return nil, fmt.Errorf("target %q: bad port %q", s, hp[1])
		}
		port = p
	}
	if uname == "" || host == "" {
		return nil, fmt.Errorf("target %q: empty user or host", s)
	}
	return &RemoteTarget{User: uname, Host: host, Port: port, Path: path}, nil
}

func currentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return os.Getenv("USER")
}

// FetchRemoteEnv reads an env file over SSH and parses it.
// Read-only: only runs `cat` on the remote path.
func FetchRemoteEnv(target string, timeout time.Duration, insecure bool) (*EnvFile, error) {
	t, err := ParseRemoteTarget(target, currentUser(), 22)
	if err != nil {
		return nil, err
	}
	client, err := sshClient(t, timeout, insecure)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	cmd := "cat " + shellQuote(t.Path)
	var out []byte
	var runErr error
	done := make(chan struct{})
	go func() {
		out, runErr = sess.CombinedOutput(cmd)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		sess.Close()
		return nil, fmt.Errorf("command timed out after %v", timeout)
	}
	if runErr != nil {
		return nil, fmt.Errorf("remote: %v: %s", runErr, strings.TrimSpace(string(out)))
	}
	f := ParseEnv(string(out))
	f.Name = target
	f.Path = target
	return f, nil
}

// sshClient dials the remote host using keys from ~/.ssh and ssh-agent.
func sshClient(t *RemoteTarget, timeout time.Duration, insecure bool) (*ssh.Client, error) {
	cb, err := hostKeyCallback(t.Host, insecure)
	if err != nil {
		return nil, err
	}
	var auths []ssh.AuthMethod
	if signers := loadKeySigners(); len(signers) > 0 {
		auths = append(auths, ssh.PublicKeys(signers...))
	}
	var agentConn net.Conn
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if conn, err := net.Dial("unix", sock); err == nil {
			agentConn = conn
			auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
		}
	}
	if len(auths) == 0 {
		return nil, fmt.Errorf("no SSH auth method available (looked in ~/.ssh/id_* and ssh-agent)")
	}
	addr := net.JoinHostPort(t.Host, strconv.Itoa(t.Port))
	cfg := &ssh.ClientConfig{
		User:            t.User,
		Auth:            auths,
		HostKeyCallback: cb,
		Timeout:         timeout,
	}
	client, err := ssh.Dial("tcp", addr, cfg)
	if agentConn != nil {
		agentConn.Close() // auth already done, agent not needed anymore
	}
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	return client, nil
}

// hostKeyCallback verifies against ~/.ssh/known_hosts by default.
// --insecure-ssh opts out explicitly.
func hostKeyCallback(host string, insecure bool) (ssh.HostKeyCallback, error) {
	if insecure {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	khPath := filepath.Join(home, ".ssh", "known_hosts")
	if _, err := os.Stat(khPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(khPath), 0o700); err != nil {
			return nil, err
		}
		f, err := os.Create(khPath)
		if err != nil {
			return nil, err
		}
		f.Close()
	}
	cb, err := knownhosts.New(khPath)
	if err != nil {
		return nil, err
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := cb(hostname, remote, key); err != nil {
			return fmt.Errorf("%v (fix: ssh-keyscan -H %s >> ~/.ssh/known_hosts)", err, host)
		}
		return nil
	}, nil
}

// loadKeySigners reads the usual private keys from ~/.ssh, skipping any
// that fail to parse (e.g. passphrase-protected keys).
func loadKeySigners() []ssh.Signer {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var out []ssh.Signer
	for _, name := range []string{"id_ed25519", "id_rsa", "id_ecdsa", "id_dsa"} {
		p, err := os.ReadFile(filepath.Join(home, ".ssh", name))
		if err != nil {
			continue
		}
		s, err := ssh.ParsePrivateKey(p)
		if err != nil {
			continue
		}
		out = append(out, s)
	}
	return out
}

// shellQuote single-quotes a string for remote shell use.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}