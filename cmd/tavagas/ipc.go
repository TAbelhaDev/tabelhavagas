package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/TAbelhaDev/tabelhascaff/ipc"
)

// postingJSON is the wire format for the ipc subcommand — Job's own fields,
// minus the ones only useful for scoring/dedupe internals (raw, description,
// type, remote).
type postingJSON struct {
	Title    string   `json:"title"`
	Company  string   `json:"company"`
	Location string   `json:"location"`
	Score    int      `json:"score"`
	Tags     []string `json:"tags"`
	Deadline string   `json:"deadline,omitempty"`
	URL      string   `json:"url"`
	Notified bool     `json:"notified"`
}

func (j Job) toIPC() postingJSON {
	return postingJSON{
		Title:    j.Title,
		Company:  j.Company,
		Location: j.Location,
		Score:    j.Score,
		Tags:     j.Tags,
		Deadline: j.Deadline,
		URL:      j.URL,
		Notified: j.Notified,
	}
}

// runIPC implements `tvagas ipc <método> [key=value...] --json`, the same
// scriptable-data-source convention as taradar/tajobs/tabelhaselfdoc — meant
// for an LLM (or any script) to ask about postings without going through
// the TUI.
func runIPC(args []string) int {
	parsed, err := ipc.ParseIPCArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "uso: tvagas ipc <método> [key=value...] --json")
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	defer store.close()

	switch parsed.Method {
	case "postings.top":
		return ipcPostingsTop(store, parsed.Filters)
	case "postings.list":
		return ipcPostingsList(store)
	default:
		fmt.Fprintf(os.Stderr, "método desconhecido: %q\n", parsed.Method)
		return 1
	}
}

// ipcPostingsTop mirrors `tvagas top`: n= how many postings (default 10),
// min= overrides the minimum score, profile= picks whose built-in/custom
// MinScore applies when min isn't given explicitly. Vetoed postings are
// excluded, same as the CLI's top.
func ipcPostingsTop(store *Store, filters map[string]string) int {
	n := 10
	if v, ok := filters["n"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil {
			n = parsed
		}
	}

	minScore := 0
	if profile, ok := filters["profile"]; ok {
		minScore = resolveProfile(profile).MinScore
	}
	if v, ok := filters["min"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil {
			minScore = parsed
		}
	}

	jobs, err := store.topFiltered(n, minScore, false, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	return ipc.WriteJSON(postingsToIPC(jobs))
}

// ipcPostingsList returns every stored posting that hasn't been vetoed.
func ipcPostingsList(store *Store) int {
	jobs, err := store.all()
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	out := make([]postingJSON, 0, len(jobs))
	for _, j := range jobs {
		if j.Vetoed {
			continue
		}
		out = append(out, j.toIPC())
	}
	return ipc.WriteJSON(out)
}

func postingsToIPC(jobs []Job) []postingJSON {
	out := make([]postingJSON, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, j.toIPC())
	}
	return out
}
