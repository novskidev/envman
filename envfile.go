package main

import (
	"bufio"
	"os"
	"strings"
)

// EnvFile is a parsed .env file.
type EnvFile struct {
	Name  string            // display name (path or remote target)
	Path  string
	Vars  map[string]string // key -> value ("" means empty)
	Order []string          // insertion order, for stable output
}

// ParseEnv parses .env content. Comments and blank lines are skipped,
// "export " prefixes are stripped, surrounding quotes are removed,
// and duplicate keys keep the last value.
func ParseEnv(content string) *EnvFile {
	f := &EnvFile{Vars: make(map[string]string)}
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue // malformed line, skip
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" {
			continue
		}
		val := strings.TrimSpace(line[eq+1:])
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if _, seen := f.Vars[key]; !seen {
			f.Order = append(f.Order, key)
		}
		f.Vars[key] = val
	}
	return f
}

// LoadEnv reads a .env file from disk.
func LoadEnv(path, name string) (*EnvFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f := ParseEnv(string(data))
	f.Path = path
	if name == "" {
		name = path
	}
	f.Name = name
	return f, nil
}