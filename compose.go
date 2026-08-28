package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// composeService holds the environment section of one compose service.
type composeService struct {
	Name        string
	Environment []string // "KEY" or "KEY=value"
}

// ComposeEnv extracts environment variables declared in a compose file.
// Supports the map form (KEY: value) and list form (- KEY=value).
// Values are not resolved — only keys matter for sync checking.
func ComposeEnv(path string) ([]composeService, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Services map[string]struct {
			Environment yaml.Node `yaml:"environment"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var out []composeService
	for name, svc := range doc.Services {
		if svc.Environment.Kind == 0 {
			continue // no environment section
		}
		env, err := envFromNode(&svc.Environment)
		if err != nil {
			return nil, fmt.Errorf("%s: environment: %w", name, err)
		}
		out = append(out, composeService{Name: name, Environment: env})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// envFromNode decodes an environment node in either map or list form.
func envFromNode(n *yaml.Node) ([]string, error) {
	var env []string
	switch n.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			key := n.Content[i].Value
			val := n.Content[i+1].Value
			if val == "" {
				env = append(env, key) // host env passthrough
			} else {
				env = append(env, key+"="+val)
			}
		}
	case yaml.SequenceNode:
		for _, item := range n.Content {
			env = append(env, item.Value)
		}
	default:
		return nil, fmt.Errorf("expected mapping or sequence, got kind %d", n.Kind)
	}
	sort.Strings(env)
	return env, nil
}

// keyOfEnvEntry returns the variable name of a "KEY" or "KEY=value" entry.
func keyOfEnvEntry(entry string) string {
	if i := strings.Index(entry, "="); i >= 0 {
		return entry[:i]
	}
	return entry
}

// ComposeReport prints per-service env summaries to w.
func ComposeReport(w io.Writer, path string, services []composeService) {
	fmt.Fprintf(w, "compose: %s\n", path)
	for _, svc := range services {
		fmt.Fprintf(w, "  %-24s %d env var(s)", svc.Name, len(svc.Environment))
		if len(svc.Environment) > 0 {
			keys := make([]string, len(svc.Environment))
			for i, e := range svc.Environment {
				keys[i] = keyOfEnvEntry(e)
			}
			fmt.Fprintf(w, ": %s", strings.Join(keys, ", "))
		}
		fmt.Fprintln(w)
	}
}