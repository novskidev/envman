package main

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// Status is a variable verdict.
type Status int

const (
	StatusOK Status = iota
	StatusMissing
	StatusEmpty
	StatusExtra
)

func (s Status) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusMissing:
		return "MISSING"
	case StatusEmpty:
		return "EMPTY"
	case StatusExtra:
		return "EXTRA"
	}
	return "?"
}

// Issue is one variable's verdict when comparing reference vs actual.
type Issue struct {
	Key    string
	Status Status
}

// Diff compares reference (.env.example) against actual (.env).
// Issues are ordered by severity then key.
func Diff(ref, actual *EnvFile) []Issue {
	var issues []Issue
	for _, key := range actual.Order {
		if _, inRef := ref.Vars[key]; !inRef {
			issues = append(issues, Issue{key, StatusExtra})
		}
	}
	for _, key := range ref.Order {
		v, inActual := actual.Vars[key]
		switch {
		case !inActual:
			issues = append(issues, Issue{key, StatusMissing})
		case strings.TrimSpace(v) == "":
			issues = append(issues, Issue{key, StatusEmpty})
		}
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Status != issues[j].Status {
			return issues[i].Status < issues[j].Status
		}
		return issues[i].Key < issues[j].Key
	})
	return issues
}

// filterOut returns issues without the given status.
func filterOut(issues []Issue, s Status) []Issue {
	out := make([]Issue, 0, len(issues))
	for _, i := range issues {
		if i.Status != s {
			out = append(out, i)
		}
	}
	return out
}

// RenderIssues prints one line per issue plus a summary line.
func RenderIssues(w io.Writer, ref, actual *EnvFile, issues []Issue, ci bool) {
	if !ci {
		fmt.Fprintf(w, "ref:     %s (%d vars)\n", ref.Name, len(ref.Vars))
		fmt.Fprintf(w, "compare: %s (%d vars)\n\n", actual.Name, len(actual.Vars))
	}
	counts := map[Status]int{}
	for _, iss := range issues {
		counts[iss.Status]++
		fmt.Fprintf(w, "%-8s %s\n", iss.Status.String()+":", iss.Key)
	}
	ok := 0
	for _, k := range ref.Order {
		if v, exists := actual.Vars[k]; exists && strings.TrimSpace(v) != "" {
			ok++
		}
	}
	fmt.Fprintf(w, "OK: %d  MISSING: %d  EMPTY: %d  EXTRA: %d\n", ok, counts[StatusMissing], counts[StatusEmpty], counts[StatusExtra])
}

// CompareTable prints a presence matrix across env files and returns the
// number of problem cells (MISSING or EMPTY).
func CompareTable(w io.Writer, files []*EnvFile) int {
	keys := map[string]bool{}
	for _, f := range files {
		for _, k := range f.Order {
			keys[k] = true
		}
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	names := make([]string, len(files))
	colW := make([]int, len(files))
	for i, f := range files {
		names[i] = filepath.Base(f.Name)
		colW[i] = len(names[i])
	}
	keyW := len("Variable")
	for _, k := range sorted {
		if len(k) > keyW {
			keyW = len(k)
		}
	}
	cells := make([][]string, len(files))
	for i := range files {
		cells[i] = make([]string, len(sorted))
	}
	issues := 0
	for j, k := range sorted {
		for i, f := range files {
			cell := "OK"
			v, ok := f.Vars[k]
			if !ok {
				cell = "MISSING"
				issues++
			} else if strings.TrimSpace(v) == "" {
				cell = "EMPTY"
				issues++
			}
			cells[i][j] = cell
			if len(cell) > colW[i] {
				colW[i] = len(cell)
			}
		}
	}
	fmt.Fprintf(w, "%-*s", keyW, "Variable")
	for i, n := range names {
		fmt.Fprintf(w, "  %-*s", colW[i], n)
	}
	fmt.Fprintln(w)
	for j, k := range sorted {
		fmt.Fprintf(w, "%-*s", keyW, k)
		for i := range files {
			fmt.Fprintf(w, "  %-*s", colW[i], cells[i][j])
		}
		fmt.Fprintln(w)
	}
	return issues
}