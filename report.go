package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Report is a machine-readable check result.
type Report struct {
	Ref      string  `json:"ref"`
	Actual   string  `json:"actual"`
	OK       int     `json:"ok"`
	Missing  int     `json:"missing"`
	Empty    int     `json:"empty"`
	Extra    int     `json:"extra"`
	ExitCode int     `json:"exit_code"`
	Issues   []Issue `json:"issues"`
}

// buildReport converts diff issues + counts into a Report.
func buildReport(ref, actual *EnvFile, issues []Issue) Report {
	r := Report{Ref: ref.Name, Actual: actual.Name, Issues: issues}
	ok := 0
	for _, k := range ref.Order {
		if v, exists := actual.Vars[k]; exists && strings.TrimSpace(v) != "" {
			ok++
		}
	}
	for _, iss := range issues {
		switch iss.Status {
		case StatusMissing:
			r.Missing++
		case StatusEmpty:
			r.Empty++
		case StatusExtra:
			r.Extra++
		}
	}
	r.OK = ok
	r.ExitCode = 1
	if r.Missing == 0 && r.Empty == 0 && r.Extra == 0 {
		r.ExitCode = 0
	}
	return r
}

// WriteJSON emits the report as JSON.
func (r Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteMarkdown emits the report as a Markdown summary (for messages/notifications).
func (r Report) WriteMarkdown(w io.Writer) {
	fmt.Fprintf(w, "## envman check: %s vs %s\n\n", r.Actual, r.Ref)
	fmt.Fprintf(w, "| Status | Count |\n|---|---|\n")
	fmt.Fprintf(w, "| :white_check_mark: OK | %d |\n", r.OK)
	fmt.Fprintf(w, "| :x: MISSING | %d |\n", r.Missing)
	fmt.Fprintf(w, "| :warning: EMPTY | %d |\n", r.Empty)
	fmt.Fprintf(w, "| :grey_exclamation: EXTRA | %d |\n", r.Extra)
	if len(r.Issues) > 0 {
		fmt.Fprintf(w, "\n**Issues:**\n\n")
		for _, iss := range r.Issues {
			fmt.Fprintf(w, "- `%s`: %s\n", iss.Key, iss.Status.String())
		}
	}
	fmt.Fprintf(w, "\nExit code: `%d`\n", r.ExitCode)
}