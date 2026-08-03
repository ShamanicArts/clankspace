package main

import (
	"bufio"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultListen = "127.0.0.1:4180"

type config struct {
	Root        string
	OmegaRoot   string
	JournalPath string
	DeployPath  string
}

type status struct {
	GeneratedAt   time.Time           `json:"generatedAt"`
	Shift         shiftSummary        `json:"shift"`
	Deployments   []deployment        `json:"deployments"`
	Gates         []gateSummary       `json:"gates"`
	Episodes      []episodeSummary    `json:"episodes"`
	Runs          []runSummary        `json:"runs"`
	ClaimsProven  []string            `json:"claimsProven"`
	ClaimsOpen    []string            `json:"claimsOpen"`
	CollectionErr []collectionWarning `json:"collectionWarnings,omitempty"`
}

type shiftSummary struct {
	State       string         `json:"state"`
	Headline    string         `json:"headline"`
	StartedAt   *time.Time     `json:"startedAt,omitempty"`
	LastUpdated *time.Time     `json:"lastUpdated,omitempty"`
	Entries     []journalEntry `json:"entries"`
}

type journalEntry struct {
	ID       string            `json:"id"`
	At       time.Time         `json:"at"`
	Kind     string            `json:"kind"`
	State    string            `json:"state,omitempty"`
	Title    string            `json:"title"`
	Body     string            `json:"body,omitempty"`
	Links    map[string]string `json:"links,omitempty"`
	Evidence map[string]string `json:"evidence,omitempty"`
}

type deployment struct {
	Name         string    `json:"name"`
	Purpose      string    `json:"purpose,omitempty"`
	Origin       string    `json:"origin,omitempty"`
	Commit       string    `json:"commit,omitempty"`
	BinarySHA256 string    `json:"binarySha256,omitempty"`
	Health       string    `json:"health"`
	Readiness    string    `json:"readiness"`
	ObservedAt   time.Time `json:"observedAt"`
}

type deploymentFile struct {
	Deployments []deployment `json:"deployments"`
}

type gateSummary struct {
	ID                   string    `json:"id"`
	Status               string    `json:"status"`
	Verdict              string    `json:"verdict,omitempty"`
	ProductionWasTouched bool      `json:"productionWasTouched"`
	UpdatedAt            time.Time `json:"updatedAt"`
	SupportedClaimCount  int       `json:"supportedClaimCount"`
	UnresolvedClaimCount int       `json:"unresolvedClaimCount"`
	PrimaryScore         *float64  `json:"primaryScore,omitempty"`
	ProjectIsolationPass *bool     `json:"projectIsolationPassed,omitempty"`
	Source               string    `json:"source"`
}

type runSummary struct {
	ID          string         `json:"id"`
	Workflow    string         `json:"workflow"`
	Status      string         `json:"status"`
	Phase       string         `json:"phase,omitempty"`
	StartedAt   *time.Time     `json:"startedAt,omitempty"`
	UpdatedAt   *time.Time     `json:"updatedAt,omitempty"`
	AgentCounts map[string]int `json:"agentCounts"`
	Workforce   []string       `json:"workforce,omitempty"`
	Outcome     string         `json:"outcome,omitempty"`
	Score       *float64       `json:"score,omitempty"`
}

type episodeSummary struct {
	ID              string     `json:"id"`
	Kind            string     `json:"kind"`
	Scenario        string     `json:"scenario"`
	Status          string     `json:"status"`
	Stage           string     `json:"stage"`
	BarrierObserved bool       `json:"barrierObserved"`
	LaneAStarted    bool       `json:"laneAStarted"`
	LaneBStarted    bool       `json:"laneBStarted"`
	Score           *float64   `json:"score,omitempty"`
	UpdatedAt       *time.Time `json:"updatedAt,omitempty"`
}

type collectionWarning struct {
	Source string `json:"source"`
	Error  string `json:"error"`
}

type event struct {
	Type       string `json:"type"`
	Status     string `json:"status"`
	RunID      string `json:"runId"`
	Workflow   string `json:"workflowFile"`
	Title      string `json:"title"`
	PhaseTitle string `json:"phaseTitle"`
	Label      string `json:"label"`
	State      string `json:"state"`
	T          int64  `json:"t"`
}

type controllerEvent struct {
	At      time.Time `json:"at"`
	Type    string    `json:"type"`
	LaneID  string    `json:"laneId"`
	Message string    `json:"message"`
}

func main() {
	if len(os.Args) < 2 {
		fatal(errors.New("usage: clank-ops serve|post"))
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = serve(os.Args[2:])
	case "post":
		err = post(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q; use serve or post", os.Args[1])
	}
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func serve(args []string) error {
	f := flag.NewFlagSet("serve", flag.ContinueOnError)
	root := f.String("root", ".", "ClankSpace checkout or runner root")
	omega := f.String("omega-root", filepath.Join(userHome(), ".omegacode", "runs"), "OmegaCode runs directory")
	journal := f.String("journal", "", "append-only shift journal path")
	deployments := f.String("deployments", "", "deployment observation JSON path")
	listen := f.String("listen", defaultListen, "loopback listen address")
	if err := f.Parse(args); err != nil {
		return err
	}
	absRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	cfg := config{Root: absRoot, OmegaRoot: *omega}
	cfg.JournalPath = pathOrDefault(*journal, filepath.Join(absRoot, "data", "ops", "shift.jsonl"))
	cfg.DeployPath = pathOrDefault(*deployments, filepath.Join(absRoot, "data", "ops", "deployments.json"))

	tpl, err := template.New("ops").Parse(indexHTML)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if err := tpl.Execute(w, map[string]string{"Title": "ClankSpace operations"}); err != nil {
			slog.Error("render operations page", "error", err)
		}
	})
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, _ *http.Request) {
		out := collect(cfg)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok"}`+"\n")
	})

	s := &http.Server{Addr: *listen, Handler: securityHeaders(mux), ReadHeaderTimeout: 5 * time.Second}
	slog.Info("clank operations ready", "listen", *listen, "root", cfg.Root, "omegaRoot", cfg.OmegaRoot, "journal", cfg.JournalPath)
	return s.ListenAndServe()
}

func post(args []string) error {
	f := flag.NewFlagSet("post", flag.ContinueOnError)
	journal := f.String("journal", "", "append-only shift journal path")
	kind := f.String("kind", "update", "update, validation, deployment, intervention, blocker, or handoff")
	state := f.String("state", "", "shift state after this entry")
	title := f.String("title", "", "short update title")
	body := f.String("body", "", "bounded update detail")
	links := stringList{}
	evidence := stringList{}
	f.Var(&links, "link", "allow-listed label=URL link; repeatable")
	f.Var(&evidence, "evidence", "allow-listed label=value evidence; repeatable")
	if err := f.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*journal) == "" || strings.TrimSpace(*title) == "" {
		return errors.New("--journal and --title are required")
	}
	if len(*title) > 180 || len(*body) > 4000 {
		return errors.New("journal title or body exceeds the bounded operations format")
	}
	entry := journalEntry{
		ID:       "ops_" + randomHex(8),
		At:       time.Now().UTC(),
		Kind:     strings.TrimSpace(*kind),
		State:    strings.TrimSpace(*state),
		Title:    strings.TrimSpace(*title),
		Body:     strings.TrimSpace(*body),
		Links:    links.mapValues(),
		Evidence: evidence.mapValues(),
	}
	return appendJournal(*journal, entry)
}

func collect(cfg config) status {
	out := status{
		GeneratedAt:   time.Now().UTC(),
		Deployments:   []deployment{},
		Gates:         []gateSummary{},
		Episodes:      []episodeSummary{},
		Runs:          []runSummary{},
		ClaimsProven:  []string{},
		ClaimsOpen:    []string{},
		CollectionErr: []collectionWarning{},
	}
	entries, err := readJournal(cfg.JournalPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		out.CollectionErr = append(out.CollectionErr, collectionWarning{Source: "journal", Error: err.Error()})
	}
	out.Shift = summarizeShift(entries)
	deployments, err := readDeployments(cfg.DeployPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		out.CollectionErr = append(out.CollectionErr, collectionWarning{Source: "deployments", Error: err.Error()})
	}
	out.Deployments = deployments
	gates, supported, open, err := readGates(filepath.Join(cfg.Root, "evals", "gates"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		out.CollectionErr = append(out.CollectionErr, collectionWarning{Source: "gates", Error: err.Error()})
	}
	out.Gates, out.ClaimsProven, out.ClaimsOpen = gates, supported, open
	episodes, err := readEpisodes(filepath.Join(cfg.Root, "data", "corpora"), 20)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		out.CollectionErr = append(out.CollectionErr, collectionWarning{Source: "episodes", Error: err.Error()})
	}
	out.Episodes = episodes
	runs, err := readRuns(cfg.OmegaRoot, 30)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		out.CollectionErr = append(out.CollectionErr, collectionWarning{Source: "omegacode", Error: err.Error()})
	}
	out.Runs = runs
	if out.Deployments == nil {
		out.Deployments = []deployment{}
	}
	if out.Gates == nil {
		out.Gates = []gateSummary{}
	}
	if out.Episodes == nil {
		out.Episodes = []episodeSummary{}
	}
	if out.Runs == nil {
		out.Runs = []runSummary{}
	}
	if out.ClaimsProven == nil {
		out.ClaimsProven = []string{}
	}
	if out.ClaimsOpen == nil {
		out.ClaimsOpen = []string{}
	}
	if out.CollectionErr == nil {
		out.CollectionErr = []collectionWarning{}
	}
	return out
}

func summarizeShift(entries []journalEntry) shiftSummary {
	s := shiftSummary{State: "not-started", Headline: "No shift update has been posted.", Entries: entries}
	if len(entries) == 0 {
		return s
	}
	first, last := entries[0].At, entries[len(entries)-1].At
	s.StartedAt, s.LastUpdated = &first, &last
	s.Headline = entries[len(entries)-1].Title
	s.State = "active"
	for _, entry := range entries {
		if entry.State != "" {
			s.State = entry.State
		}
	}
	return s
}

func readJournal(path string) ([]journalEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var entries []journalEntry
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 16*1024), 256*1024)
	for s.Scan() {
		var entry journalEntry
		if err := json.Unmarshal(s.Bytes(), &entry); err != nil {
			return nil, fmt.Errorf("decode journal: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, s.Err()
}

func appendJournal(path string, entry journalEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err = f.Write(append(b, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func readDeployments(path string) ([]deployment, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f deploymentFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return f.Deployments, nil
}

func readGates(dir string) ([]gateSummary, []string, []string, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.result.json"))
	if err != nil {
		return nil, nil, nil, err
	}
	var gates []gateSummary
	var latestSupported, latestOpen []string
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, nil, err
		}
		var raw map[string]any
		if err := json.Unmarshal(b, &raw); err != nil {
			return nil, nil, nil, fmt.Errorf("decode %s: %w", filepath.Base(path), err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, nil, nil, err
		}
		supported := stringSlice(raw["claimsSupported"])
		open := stringSlice(raw["claimsNotYetEstablished"])
		g := gateSummary{
			ID:                   stringValue(raw["gateId"], strings.TrimSuffix(filepath.Base(path), ".result.json")),
			Status:               stringValue(raw["status"], "unknown"),
			Verdict:              stringValue(raw["productVerdict"], ""),
			ProductionWasTouched: boolValue(raw["productionWasTouched"]),
			UpdatedAt:            info.ModTime().UTC(),
			SupportedClaimCount:  len(supported),
			UnresolvedClaimCount: len(open),
			Source:               filepath.ToSlash(filepath.Join("evals", "gates", filepath.Base(path))),
		}
		if iso, ok := raw["projectIsolationProbe"].(map[string]any); ok {
			v := boolValue(iso["passed"])
			g.ProjectIsolationPass = &v
		}
		if overlap, ok := raw["alignedOverlapRegression"].(map[string]any); ok {
			if fixed, ok := overlap["fixedEpisode"].(map[string]any); ok {
				if score, ok := numberValue(fixed["score"]); ok {
					g.PrimaryScore = &score
				}
			}
		}
		gates = append(gates, g)
		latestSupported, latestOpen = supported, open
	}
	sort.Slice(gates, func(i, j int) bool { return gates[i].ID > gates[j].ID })
	if len(gates) > 0 {
		latestID := gates[0].ID
		for _, path := range paths {
			b, _ := os.ReadFile(path)
			var raw map[string]any
			_ = json.Unmarshal(b, &raw)
			if stringValue(raw["gateId"], "") == latestID {
				latestSupported = stringSlice(raw["claimsSupported"])
				latestOpen = stringSlice(raw["claimsNotYetEstablished"])
				break
			}
		}
	}
	return gates, latestSupported, latestOpen, nil
}

func readRuns(root string, limit int) ([]runSummary, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var dirs []os.DirEntry
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "wf_") {
			dirs = append(dirs, entry)
		}
	}
	sort.Slice(dirs, func(i, j int) bool {
		a, _ := dirs[i].Info()
		b, _ := dirs[j].Info()
		return a.ModTime().After(b.ModTime())
	})
	if len(dirs) > limit {
		dirs = dirs[:limit]
	}
	runs := make([]runSummary, 0, len(dirs))
	for _, dir := range dirs {
		run, err := readRun(filepath.Join(root, dir.Name()), dir.Name())
		if err != nil {
			continue
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func readEpisodes(root string, limit int) ([]episodeSummary, error) {
	if _, err := os.Stat(root); err != nil {
		return nil, err
	}
	var controllerPaths []string
	var rolloutDirs []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && entry.Name() == "controller-events.jsonl" {
			controllerPaths = append(controllerPaths, path)
		}
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "episode_") && filepath.Base(filepath.Dir(path)) == "traces" {
			rolloutDirs = append(rolloutDirs, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var episodes []episodeSummary
	for _, path := range controllerPaths {
		episode, err := readEpisode(path, root)
		if err == nil {
			episodes = append(episodes, episode)
		}
	}
	for _, dir := range rolloutDirs {
		episode, err := readRolloutEpisode(dir, root)
		if err == nil {
			episodes = append(episodes, episode)
		}
	}
	sort.Slice(episodes, func(i, j int) bool {
		if episodes[i].UpdatedAt == nil {
			return false
		}
		if episodes[j].UpdatedAt == nil {
			return true
		}
		return episodes[i].UpdatedAt.After(*episodes[j].UpdatedAt)
	})
	if len(episodes) > limit {
		episodes = episodes[:limit]
	}
	return episodes, nil
}

func readEpisode(eventsPath, root string) (episodeSummary, error) {
	episodeDir := filepath.Dir(eventsPath)
	rel, _ := filepath.Rel(root, episodeDir)
	parts := strings.Split(filepath.ToSlash(rel), "/")
	e := episodeSummary{ID: filepath.Base(episodeDir), Kind: "collaboration", Status: "running", Stage: "starting"}
	for i, part := range parts {
		if part == "traces" && i >= 2 {
			e.Scenario = parts[i-2]
			break
		}
	}
	f, err := os.Open(eventsPath)
	if err != nil {
		return e, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var item controllerEvent
		if json.Unmarshal(scanner.Bytes(), &item) != nil {
			continue
		}
		at := item.At.UTC()
		e.UpdatedAt = &at
		switch item.Type {
		case "episode.started":
			e.Stage = "episode started"
		case "lane.started":
			if item.LaneID == "lane-a" {
				e.LaneAStarted = true
				e.Stage = "lane A running"
			} else if item.LaneID == "lane-b" {
				e.LaneBStarted = true
				e.Stage = "lane B running"
			}
		case "barrier.candidate":
			e.Stage = "checkpoint candidate observed"
		case "barrier.observed":
			e.BarrierObserved = true
			e.Stage = "barrier passed"
		case "episode.finished":
			e.Status = "completed"
			e.Stage = "completed"
		}
	}
	if err := scanner.Err(); err != nil {
		return e, err
	}
	if b, err := os.ReadFile(filepath.Join(episodeDir, "collaboration.json")); err == nil {
		var result struct {
			ScenarioID string  `json:"scenarioId"`
			Status     string  `json:"status"`
			Score      float64 `json:"score"`
		}
		if json.Unmarshal(b, &result) == nil {
			if result.ScenarioID != "" {
				e.Scenario = result.ScenarioID
			}
			if result.Status != "" {
				e.Status = result.Status
			}
			e.Score = &result.Score
		}
	}
	if e.Status == "running" && e.UpdatedAt != nil && time.Since(*e.UpdatedAt) > 30*time.Minute {
		e.Status = "incomplete"
		e.Stage = "stale after " + e.Stage
	}
	return e, nil
}

func readRolloutEpisode(episodeDir, root string) (episodeSummary, error) {
	rel, _ := filepath.Rel(root, episodeDir)
	parts := strings.Split(filepath.ToSlash(rel), "/")
	e := episodeSummary{ID: filepath.Base(episodeDir), Kind: "single-agent", Status: "running", Stage: "starting"}
	for i, part := range parts {
		if part == "traces" && i >= 2 {
			e.Scenario = parts[i-2]
			break
		}
	}
	entries, err := os.ReadDir(episodeDir)
	if err != nil {
		return e, err
	}
	turns, completedTurns := 0, 0
	var newest time.Time
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "turn-") {
			continue
		}
		turns++
		turnDir := filepath.Join(episodeDir, entry.Name())
		if _, err := os.Stat(filepath.Join(turnDir, "response.txt")); err == nil {
			completedTurns++
		}
		_ = filepath.WalkDir(turnDir, func(path string, item os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if info, infoErr := item.Info(); infoErr == nil && info.ModTime().After(newest) {
				newest = info.ModTime()
			}
			return nil
		})
	}
	if turns > 0 {
		e.Stage = fmt.Sprintf("turn %d running", turns)
	}
	if turns > 0 && completedTurns == turns {
		e.Stage = "finalizing"
	}
	if b, err := os.ReadFile(filepath.Join(episodeDir, "rollout.json")); err == nil {
		var result struct {
			EpisodeID  string    `json:"episodeId"`
			ScenarioID string    `json:"scenarioId"`
			EndedAt    time.Time `json:"endedAt"`
		}
		if json.Unmarshal(b, &result) == nil {
			e.Status = "completed"
			e.Stage = "completed"
			if result.EpisodeID != "" {
				e.ID = result.EpisodeID
			}
			if result.ScenarioID != "" {
				e.Scenario = result.ScenarioID
			}
			if !result.EndedAt.IsZero() {
				newest = result.EndedAt
			}
		}
	}
	if !newest.IsZero() {
		at := newest.UTC()
		e.UpdatedAt = &at
	}
	if e.Status == "running" && e.UpdatedAt != nil && time.Since(*e.UpdatedAt) > 30*time.Minute {
		e.Status = "incomplete"
		e.Stage = "stale after " + e.Stage
	}
	return e, nil
}

func readRun(path, id string) (runSummary, error) {
	f, err := os.Open(filepath.Join(path, "events.jsonl"))
	if err != nil {
		return runSummary{}, err
	}
	defer f.Close()
	r := runSummary{ID: id, Status: "unknown", AgentCounts: map[string]int{}}
	agentStates := map[string]string{}
	workforce := map[string]struct{}{}
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 16*1024), 2*1024*1024)
	for s.Scan() {
		var e event
		if json.Unmarshal(s.Bytes(), &e) != nil {
			continue
		}
		at := time.UnixMilli(e.T).UTC()
		if r.UpdatedAt == nil || at.After(*r.UpdatedAt) {
			r.UpdatedAt = &at
		}
		switch e.Type {
		case "run":
			if e.Status != "" {
				r.Status = e.Status
			}
			if e.Workflow != "" {
				r.Workflow = filepath.Base(e.Workflow)
			}
			if e.Status == "started" && r.StartedAt == nil {
				t := at
				r.StartedAt = &t
			}
		case "phase":
			if e.Title != "" {
				r.Phase = e.Title
			}
		case "agent":
			key := e.Label
			if key == "" {
				key = strconv.Itoa(len(agentStates) + 1)
			}
			if e.State != "" {
				agentStates[key] = e.State
			}
			if e.Label != "" {
				workforce[e.Label] = struct{}{}
			}
		}
	}
	if err := s.Err(); err != nil {
		return runSummary{}, err
	}
	for _, state := range agentStates {
		r.AgentCounts[state]++
	}
	for label := range workforce {
		r.Workforce = append(r.Workforce, label)
	}
	sort.Strings(r.Workforce)
	if b, err := os.ReadFile(filepath.Join(path, "result.json")); err == nil {
		var raw map[string]any
		if json.Unmarshal(b, &raw) == nil {
			r.Outcome, r.Score = resultOutcome(raw)
			if r.Status == "started" || r.Status == "unknown" {
				r.Status = "completed"
			}
		}
	}
	return r, nil
}

func resultOutcome(raw map[string]any) (string, *float64) {
	if verdicts, ok := raw["verdicts"].([]any); ok && len(verdicts) > 0 {
		if first, ok := verdicts[0].(map[string]any); ok {
			accepted := boolValue(first["accepted"])
			outcome := "rejected"
			if accepted {
				outcome = "accepted"
			}
			if score, ok := numberValue(first["score"]); ok {
				return outcome, &score
			}
			return outcome, nil
		}
	}
	accepted, aok := raw["accepted"].([]any)
	rejected, rok := raw["rejected"].([]any)
	if aok || rok {
		return fmt.Sprintf("%d accepted / %d rejected", len(accepted), len(rejected)), nil
	}
	return "completed", nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	if !strings.Contains(v, "=") {
		return errors.New("expected label=value")
	}
	*s = append(*s, v)
	return nil
}
func (s stringList) mapValues() map[string]string {
	if len(s) == 0 {
		return nil
	}
	out := make(map[string]string, len(s))
	for _, item := range s {
		key, value, _ := strings.Cut(item, "=")
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out
}

func pathOrDefault(got, fallback string) string {
	if strings.TrimSpace(got) == "" {
		return fallback
	}
	return got
}

func userHome() string {
	home, _ := os.UserHomeDir()
	return home
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

func stringValue(v any, fallback string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return fallback
}

func boolValue(v any) bool {
	b, _ := v.(bool)
	return b
}

func numberValue(v any) (float64, bool) {
	n, ok := v.(float64)
	return n, ok
}

func stringSlice(v any) []string {
	values, _ := v.([]any)
	out := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

//go:embed web/index.html
var indexHTML string
