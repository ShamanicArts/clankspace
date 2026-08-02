package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ShamanicArts/clankspace/internal/client"
	"github.com/ShamanicArts/clankspace/internal/config"
	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/githubsync"
	"github.com/ShamanicArts/clankspace/internal/httpapi"
	"github.com/ShamanicArts/clankspace/internal/mcpserver"
	"github.com/ShamanicArts/clankspace/internal/service"
	"github.com/ShamanicArts/clankspace/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "dev"

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "clank:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("command required")
	}
	if args[0] == "serve" {
		return serve(ctx)
	}
	if args[0] == "context" {
		return localContext()
	}
	c := client.New(value("CLANKSPACE_URL", "http://localhost:8080"), os.Getenv("CLANKSPACE_TOKEN"))
	if c.Token == "" {
		return errors.New("CLANKSPACE_TOKEN is required")
	}
	switch args[0] {
	case "version":
		fmt.Println(version)
		return nil
	case "mcp":
		return mcpserver.New(c).Run(ctx, &mcp.StdioTransport{})
	case "project":
		return project(ctx, c, args[1:])
	case "run":
		return runCommand(ctx, c, args[1:])
	case "note":
		return note(ctx, c, args[1:])
	case "trajectory":
		return trajectory(ctx, c, args[1:])
	case "brief":
		return brief(ctx, c, args[1:])
	case "why":
		whyArgs := args[1:]
		if len(whyArgs) > 0 && !strings.HasPrefix(whyArgs[0], "-") {
			whyArgs = append([]string{"--query", whyArgs[0]}, whyArgs[1:]...)
		}
		return brief(ctx, c, whyArgs)
	case "repo":
		return repo(ctx, c, args[1:])
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func serve(ctx context.Context) error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(cfg.DataDir, 0700); err != nil {
		return err
	}
	db, err := store.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer db.Close()
	p, err := db.EnsureBootstrap(ctx, cfg.BootstrapToken, cfg.WorkspaceName, cfg.OwnerName)
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	core := service.New(db)
	h := (&httpapi.Server{Store: db, Core: core, GitHub: githubsync.New(cfg.GitHubToken), Log: log}).Handler()
	srv := &http.Server{Addr: cfg.Listen, Handler: h, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	stopCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-stopCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	log.Info("clankspace ready", "listen", cfg.Listen, "database", filepath.Clean(cfg.DatabasePath), "principal", p.DisplayName, "version", version)
	err = srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func project(ctx context.Context, c *client.Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clank project list|create")
	}
	if args[0] == "list" {
		o, e := c.ListProjects(ctx)
		if e == nil {
			printJSON(o)
		}
		return e
	}
	if args[0] == "export" {
		f := flag.NewFlagSet("project export", flag.ContinueOnError)
		projectID := f.String("project", os.Getenv("CLANKSPACE_PROJECT"), "project ID or slug")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		o, err := c.ExportProject(ctx, *projectID)
		if err == nil {
			printJSON(o)
		}
		return err
	}
	if args[0] == "token" {
		f := flag.NewFlagSet("project token", flag.ContinueOnError)
		projectID := f.String("project", os.Getenv("CLANKSPACE_PROJECT"), "project ID or slug")
		name := f.String("name", "", "project agent identity name")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		o, err := c.IssueProjectToken(ctx, *projectID, *name)
		if err == nil {
			printJSON(o)
		}
		return err
	}
	if args[0] != "create" {
		return errors.New("usage: clank project list|create|export|token")
	}
	f := flag.NewFlagSet("project create", flag.ContinueOnError)
	slug := f.String("slug", "", "lowercase project slug")
	name := f.String("name", "", "project name")
	description := f.String("description", "", "project description")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	o, e := c.CreateProject(ctx, *slug, *name, *description)
	if e == nil {
		printJSON(o)
	}
	return e
}

func runCommand(ctx context.Context, c *client.Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clank run start|end")
	}
	if args[0] == "end" {
		f := flag.NewFlagSet("run end", flag.ContinueOnError)
		id := f.String("id", value("CLANKSPACE_RUN", ""), "run ID")
		outcome := f.String("outcome", "completed", "outcome")
		verification := f.String("verification", "", "verification summary")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		o, e := c.EndRun(ctx, *id, domain.EndRunInput{Outcome: *outcome, Verification: *verification})
		if e == nil {
			printJSON(o)
		}
		return e
	}
	f := flag.NewFlagSet("run start", flag.ContinueOnError)
	project := f.String("project", os.Getenv("CLANKSPACE_PROJECT"), "project ID or slug")
	agent := f.String("agent", "agent", "agent name")
	harness := f.String("harness", "", "harness")
	harnessVersion := f.String("harness-version", "", "harness version")
	provider := f.String("provider", "", "provider")
	model := f.String("model", "", "model")
	reasoning := f.String("reasoning", "", "reasoning effort")
	role := f.String("role", "primary", "primary|subagent|reviewer|automation|integration")
	runType := f.String("type", "interactive", "interactive|automation")
	objective := f.String("objective", "", "objective")
	branch := f.String("branch", "", "git branch")
	worktree := f.String("worktree", "", "worktree")
	paths := f.String("instructions", "", "instruction profile names/hashes, comma separated")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	o, e := c.StartRun(ctx, domain.StartRunInput{ProjectID: *project, AgentName: *agent, Harness: *harness, HarnessVersion: *harnessVersion, Provider: *provider, Model: *model, Reasoning: *reasoning, Role: *role, RunType: *runType, Objective: *objective, Branch: *branch, Worktree: *worktree, InstructionProfile: split(*paths)})
	if e == nil {
		printJSON(o)
	}
	return e
}

func note(ctx context.Context, c *client.Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clank note add|supersede")
	}
	if args[0] == "supersede" {
		f := flag.NewFlagSet("note supersede", flag.ContinueOnError)
		project := f.String("project", os.Getenv("CLANKSPACE_PROJECT"), "project")
		id := f.String("id", "", "note ID")
		revision := f.Int("revision", 1, "expected revision")
		reason := f.String("reason", "", "reason")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		o, e := c.SupersedeNote(ctx, *project, *id, domain.SupersedeNoteInput{RunID: os.Getenv("CLANKSPACE_RUN"), ExpectedRevision: *revision, Reason: *reason})
		if e == nil {
			printJSON(o)
		}
		return e
	}
	f := flag.NewFlagSet("note add", flag.ContinueOnError)
	project := f.String("project", os.Getenv("CLANKSPACE_PROJECT"), "project")
	runID := f.String("run", os.Getenv("CLANKSPACE_RUN"), "run ID")
	kind := f.String("kind", "intent", "intent|decision|understanding|observation|checkpoint")
	title := f.String("title", "", "title")
	summary := f.String("summary", "", "concise project implication")
	rationale := f.String("rationale", "", "reasoning summary, not chain-of-thought")
	ledBy := f.String("led-by", "agent", "human|agent|joint|external")
	basis := f.String("basis", "autonomous_agent_judgment", "direction basis")
	paths := f.String("paths", "", "comma-separated paths")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	o, e := c.CreateNote(ctx, *project, domain.CreateNoteInput{RunID: *runID, Kind: *kind, Title: *title, Summary: *summary, Rationale: *rationale, LedBy: *ledBy, DirectionBasis: *basis, Paths: split(*paths)})
	if e == nil {
		printJSON(o)
	}
	return e
}

func trajectory(ctx context.Context, c *client.Client, args []string) error {
	if len(args) == 0 || args[0] != "start" {
		return errors.New("usage: clank trajectory start")
	}
	f := flag.NewFlagSet("trajectory start", flag.ContinueOnError)
	project := f.String("project", os.Getenv("CLANKSPACE_PROJECT"), "project")
	runID := f.String("run", os.Getenv("CLANKSPACE_RUN"), "run ID")
	objective := f.String("objective", "", "objective")
	rationale := f.String("rationale", "", "reasoned intent")
	paths := f.String("paths", "", "comma-separated paths")
	branch := f.String("branch", "", "branch")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	o, e := c.CreateTrajectory(ctx, *project, domain.CreateTrajectoryInput{RunID: *runID, Objective: *objective, Rationale: *rationale, Paths: split(*paths), Branch: *branch})
	if e == nil {
		printJSON(o)
	}
	return e
}

func brief(ctx context.Context, c *client.Client, args []string) error {
	f := flag.NewFlagSet("brief", flag.ContinueOnError)
	project := f.String("project", os.Getenv("CLANKSPACE_PROJECT"), "project")
	runID := f.String("run", os.Getenv("CLANKSPACE_RUN"), "current run ID")
	query := f.String("query", "", "search terms")
	objective := f.String("objective", "", "proposed objective")
	paths := f.String("paths", "", "comma-separated paths")
	if err := f.Parse(args); err != nil {
		return err
	}
	o, e := c.Brief(ctx, *project, domain.BriefInput{RunID: *runID, Query: *query, Objective: *objective, Paths: split(*paths), CheckConflicts: true})
	if e == nil {
		printJSON(o)
	}
	return e
}
func repo(ctx context.Context, c *client.Client, args []string) error {
	if len(args) == 0 || args[0] != "attach" {
		return errors.New("usage: clank repo attach")
	}
	f := flag.NewFlagSet("repo attach", flag.ContinueOnError)
	project := f.String("project", os.Getenv("CLANKSPACE_PROJECT"), "project")
	url := f.String("url", "", "public GitHub URL")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	o, e := c.AttachRepository(ctx, *project, *url)
	if e == nil {
		printJSON(o)
	}
	return e
}

func split(s string) []string {
	var out []string
	for _, x := range strings.Split(s, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}
func localContext() error {
	type localFile struct {
		URL     string `json:"url"`
		Project string `json:"project"`
	}
	cwd, _ := os.Getwd()
	local := localFile{URL: value("CLANKSPACE_URL", "http://localhost:8080"), Project: os.Getenv("CLANKSPACE_PROJECT")}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		b, err := os.ReadFile(filepath.Join(dir, ".clankspace.json"))
		if err == nil {
			_ = json.Unmarshal(b, &local)
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = cwd
		b, err := cmd.Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
	printJSON(map[string]any{
		"url": local.URL, "project": local.Project, "repository": git("remote", "get-url", "origin"),
		"branch": git("branch", "--show-current"), "head": git("rev-parse", "HEAD"), "worktree": cwd,
		"tokenConfigured": os.Getenv("CLANKSPACE_TOKEN") != "", "notice": domain.AdvisoryNotice,
	})
	return nil
}
func printJSON(v any) { e := json.NewEncoder(os.Stdout); e.SetIndent("", "  "); _ = e.Encode(v) }
func value(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func usage() {
	fmt.Fprintln(os.Stderr, "clank serve | context | project | run | note | trajectory | brief | why | repo | mcp")
}
