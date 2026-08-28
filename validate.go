package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rule is one variable's validation rule from .envman.yaml.
type Rule struct {
	Required bool   `yaml:"required"`
	Type     string `yaml:"type"` // url, port, boolean, integer, float, or empty
	Pattern  string `yaml:"pattern"`
}

// Config is the optional .envman.yaml file.
type Config struct {
	Rules map[string]Rule `yaml:"rules"`
}

// LoadConfig reads .envman.yaml if present. Missing file is not an error.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Rules: map[string]Rule{}}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.Rules == nil {
		cfg.Rules = map[string]Rule{}
	}
	return &cfg, nil
}

// ValidationIssue is one rule violation.
type ValidationIssue struct {
	Key   string
	Field string // "required" | "type" | "pattern"
	Msg   string
}

// Validate applies config rules to an env file.
func (c *Config) Validate(f *EnvFile) []ValidationIssue {
	var out []ValidationIssue
	for key, rule := range c.Rules {
		val, exists := f.Vars[key]
		if !exists {
			if rule.Required {
				out = append(out, ValidationIssue{key, "required", "missing required variable"})
			}
			continue
		}
		if strings.TrimSpace(val) == "" {
			if rule.Required {
				out = append(out, ValidationIssue{key, "required", "required variable is empty"})
			}
			continue
		}
		if rule.Type != "" {
			if msg := checkType(rule.Type, val); msg != "" {
				out = append(out, ValidationIssue{key, "type", msg})
			}
		}
		if rule.Pattern != "" {
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				out = append(out, ValidationIssue{key, "pattern", fmt.Sprintf("bad pattern %q: %v", rule.Pattern, err)})
				continue
			}
			if !re.MatchString(val) {
				out = append(out, ValidationIssue{key, "pattern", fmt.Sprintf("value does not match %q", rule.Pattern)})
			}
		}
	}
	return out
}

func checkType(t, val string) string {
	switch t {
	case "url":
		u, err := url.Parse(val)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Sprintf("expected URL, got %q", val)
		}
	case "port":
		n, err := strconv.Atoi(val)
		if err != nil || n < 1 || n > 65535 {
			return fmt.Sprintf("expected port 1-65535, got %q", val)
		}
	case "boolean":
		if !isBool(val) {
			return fmt.Sprintf("expected boolean (true/false/1/0/yes/no), got %q", val)
		}
	case "integer":
		if _, err := strconv.Atoi(val); err != nil {
			return fmt.Sprintf("expected integer, got %q", val)
		}
	case "float":
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			return fmt.Sprintf("expected float, got %q", val)
		}
	default:
		return fmt.Sprintf("unknown type %q in .envman.yaml", t)
	}
	return ""
}

func isBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "false", "1", "0", "yes", "no", "on", "off":
		return true
	}
	return false
}

// requiredFromComments mines "# required" comments in a .env.example file.
func requiredFromComments(content string) map[string]bool {
	required := map[string]bool{}
	lines := strings.Split(content, "\n")
	// comment can precede or follow the variable line
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		marked := strings.Contains(strings.ToLower(trimmed), "required")
		if !marked {
			continue
		}
		// inline comment (KEY=... # required): flag own key, no lookahead
		if strings.Contains(trimmed, "=") {
			if key := keyOfLine(trimmed); key != "" {
				required[key] = true
			}
			continue
		}
		// comment before: KEY=... on next line
		if i+1 < len(lines) {
			key := keyOfLine(lines[i+1])
			if key != "" {
				required[key] = true
			}
		}
	}
	return required
}

func keyOfLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}
	line = strings.TrimPrefix(line, "export ")
	eq := strings.Index(line, "=")
	if eq < 0 {
		return ""
	}
	return strings.TrimSpace(line[:eq])
}

func renderValidation(w io.Writer, issues []ValidationIssue) {
	if len(issues) == 0 {
		fmt.Fprintln(w, "Validation: all rules satisfied ✓")
		return
	}
	for _, iss := range issues {
		fmt.Fprintf(w, "%-9s %s: %s\n", strings.ToUpper(iss.Field)+":", iss.Key, iss.Msg)
	}
}