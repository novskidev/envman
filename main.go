package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

const version = "0.3.0"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		usage(os.Stdout)
		return 0
	}
	switch args[0] {
	case "check":
		return cmdCheck(args[1:])
	case "compare":
		return cmdCompare(args[1:])
	case "scan":
		return cmdScan(args[1:])
	case "validate":
		return cmdValidate(args[1:])
	case "compose":
		return cmdCompose(args[1:])
	case "version", "--version", "-v":
		fmt.Printf("envman %s\n", version)
		return 0
	case "help", "--help", "-h":
		usage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "envman: unknown command %q\n\n", args[0])
		usage(os.Stderr)
		return 2
	}
}

func cmdCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	flRemote := fs.String("remote", "", "compare against remote env: user@host:/path/.env (SSH)")
	flCI := fs.Bool("ci", false, "machine-readable output; exit code is the signal")
	flExample := fs.String("example", ".env.example", "reference file")
	flEnv := fs.String("env", ".env", "local env file")
	flAllowExtra := fs.Bool("allow-extra", false, "EXTRA variables don't affect the exit code")
	flJSON := fs.Bool("json", false, "emit machine-readable JSON report")
	flMarkdown := fs.Bool("markdown", false, "emit Markdown report")
	flTimeout := fs.Duration("timeout", 10*time.Second, "SSH timeout")
	flInsecure := fs.Bool("insecure-ssh", false, "skip known_hosts verification (NOT recommended)")
	fs.Parse(args)
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "envman check: unexpected argument %q\n", fs.Arg(0))
		return 2
	}

	ref, err := LoadEnv(*flExample, *flExample)
	if err != nil {
		fmt.Fprintf(os.Stderr, "envman: local: %v\n", err)
		return 2
	}
	var act *EnvFile
	if *flRemote != "" {
		act, err = FetchRemoteEnv(*flRemote, *flTimeout, *flInsecure)
		if err != nil {
			fmt.Fprintf(os.Stderr, "envman: remote: %v\n", err)
			return 3
		}
	} else {
		act, err = LoadEnv(*flEnv, *flEnv)
		if err != nil {
			fmt.Fprintf(os.Stderr, "envman: local: %v\n", err)
			return 2
		}
	}

	tissues := Diff(ref, act)
	critical := issues
	if *flAllowExtra {
		critical = filterOut(issues, StatusExtra)
	}
	switch {
	case *flJSON:
		r := buildReport(ref, act, issues)
		if *flAllowExtra {
			r.Extra = 0
			if r.Missing == 0 && r.Empty == 0 {
				r.ExitCode = 0
			} else {
				r.ExitCode = 1
			}
		}
		if err := r.WriteJSON(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "envman: report: %v\n", err)
			return 2
		}
	case *flMarkdown:
		r := buildReport(ref, act, issues)
		r.WriteMarkdown(os.Stdout)
	default:
		RenderIssues(os.Stdout, ref, act, issues, *flCI)
	}
	if len(critical) > 0 {
		return 1
	}
	return 0
}

func cmdCompose(args []string) int {
	fs := flag.NewFlagSet("compose", flag.ExitOnError)
	flFile := fs.String("file", "docker-compose.yml", "compose file to inspect")
	fs.Parse(args)
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "envman compose: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	svcs, err := ComposeEnv(*flFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "envman: %v\n", err)
		return 2
	}
	ComposeReport(os.Stdout, *flFile, svcs)
	return 0
}

func cmdCompare(args []string) int {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	fs.Parse(args)
	files := fs.Args()
	if len(files) < 2 {
		fmt.Fprintln(os.Stderr, "envman compare: need at least 2 files")
		fmt.Fprintln(os.Stderr, "usage: envman compare .env.local .env.staging .env.production")
		return 2
	}
	parsed := make([]*EnvFile, 0, len(files))
	for _, p := range files {
		f, err := LoadEnv(p, p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "envman: %v\n", err)
			return 2
		}
		parsed = append(parsed, f)
	}
	issues := CompareTable(os.Stdout, parsed)
	if issues > 0 {
		fmt.Fprintf(os.Stderr, "%d problem(s) detected\n", issues)
		return 1
	}
	return 0
}

func cmdScan(args []string) int {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	flFile := fs.String("file", ".env.example", "env file to scan")
	fs.Parse(args)
	f, err := LoadEnv(*flFile, *flFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "envman: %v\n", err)
		return 2
	}
	findings := ScanSecrets(f)
	renderScan(os.Stdout, f, findings)
	if len(findings) > 0 {
		return 1
	}
	return 0
}

func cmdValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	flEnv := fs.String("env", ".env", "env file to validate")
	flConfig := fs.String("config", ".envman.yaml", "rules file")
	flExample := fs.String("example", ".env.example", "reference file to mine # required comments")
	fs.Parse(args)
	cfg, err := LoadConfig(*flConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "envman: %v\n", err)
		return 2
	}
	f, err := LoadEnv(*flEnv, *flEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "envman: %v\n", err)
		return 2
	}
	var issues []ValidationIssue
	issues = append(issues, cfg.Validate(f)...)
	// mine # required from .env.example and check those too
	if _, err := LoadEnv(*flExample, *flExample); err == nil {
		if content, err2 := os.ReadFile(*flExample); err2 == nil {
			for key := range requiredFromComments(string(content)) {
				if _, ok := cfg.Rules[key]; ok {
					continue // already rule-defined
				}
				cfg.Rules[key] = Rule{Required: true}
			}
			issues = append(issues, cfg.Validate(f)...)
		}
	}
	renderValidation(os.Stdout, issues)
	if len(issues) > 0 {
		return 1
	}
	return 0
}

func usage(w io.Writer) {
	fmt.Fprintf(w, `envman %s - environment variable sync checker

Usage:
  envman check [flags]
      Compare reference (.env.example) with actual env (.env).
      Detects: MISSING, EMPTY, EXTRA. Exit 1 on problems.
  envman check --remote user@host:/path/to/.env
      Same check, but the actual env is fetched over SSH (read-only).
  envman compare FILE1 FILE2 [FILE3 ...]
      Presence matrix of variables across env files.
  envman scan [--file .env.example]
      Detect values that look like real secrets in an env file
      (e.g. committed in a .env.example by accident).
  envman validate [--env .env] [--config .envman.yaml] [--example .env.example]
      Check rules from .envman.yaml + "# required" comments in .env.example.
  envman compose [--file docker-compose.yml]
      List environment variables declared per service in a compose file.

Flags (check):
  --example PATH       reference file           (default .env.example)
  --env PATH           local env file           (default .env)
  --remote TARGET      compare with remote env over SSH
  --ci                 minimal output, exit code is the signal
  --allow-extra        EXTRA vars don't affect the exit code
  --json               emit machine-readable JSON report
  --markdown           emit Markdown report
  --timeout DURATION   SSH timeout              (default 10s)
  --insecure-ssh       skip known_hosts check (not recommended)

Exit codes:
  0  all OK
  1  problems found (MISSING / EMPTY / EXTRA, unless --allow-extra)
  2  usage or local file error
  3  remote (SSH) error

Examples:
  envman check
  envman check --ci
  envman check --remote deploy@vps.example.com:/srv/app/.env --ci
  envman compare .env.local .env.staging .env.production
`, version)
}