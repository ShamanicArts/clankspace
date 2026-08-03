package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
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
	"github.com/ShamanicArts/clankspace/internal/localconfig"
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
	if isHelp(args[0]) {
		usage()
		return nil
	}
	if args[0] == "run" && len(args) > 1 && isHelp(args[1]) {
		runUsage()
		return nil
	}
	if args[0] == "note" && len(args) > 1 && (isHelp(args[1]) || (len(args) > 2 && isHelp(args[2]))) {
		return note(ctx, nil, args[1:])
	}
	if args[0] == "trajectory" && len(args) > 1 && (isHelp(args[1]) || (len(args) > 2 && isHelp(args[2]))) {
		return trajectory(ctx, nil, args[1:])
	}
	if args[0] == "serve" {
		return serve(ctx)
	}
	if args[0] == "version" {
		fmt.Println(version)
		return nil
	}
	resolved, err := localconfig.Resolve("")
	if err != nil {
		return err
	}
	if args[0] == "context" {
		return localContext(resolved)
	}
	if args[0] == "auth" {
		return auth(resolved, args[1:])
	}
	if os.Getenv("CLANKSPACE_PROJECT") == "" && resolved.Project != "" {
		if err = os.Setenv("CLANKSPACE_PROJECT", resolved.Project); err != nil {
			return err
		}
	}
	c := client.New(resolved.URL, resolved.Token)
	if c.Token == "" {
		return errors.New("no ClankSpace credential for this project; run: clank auth set --token-stdin")
	}
	switch args[0] {
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

func auth(resolved localconfig.Resolved, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clank auth set|status")
	}
	switch args[0] {
	case "status":
		printJSON(map[string]any{
			"url": resolved.URL, "project": resolved.Project,
			"tokenConfigured": resolved.Token != "", "tokenSource": resolved.TokenSource,
			"projectFile": resolved.ProjectFilePath, "credentialsFile": resolved.CredentialsPath,
		})
		return nil
	case "set":
		f := flag.NewFlagSet("auth set", flag.ContinueOnError)
		url := f.String("url", resolved.URL, "ClankSpace server URL")
		project := f.String("project", resolved.Project, "project ID or slug")
		tokenStdin := f.Bool("token-stdin", false, "read the project token from standard input")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		if !*tokenStdin {
			return errors.New("refusing token in command arguments; pipe it to: clank auth set --token-stdin")
		}
		body, err := io.ReadAll(io.LimitReader(os.Stdin, 4097))
		if err != nil {
			return err
		}
		if len(body) > 4096 {
			return errors.New("token exceeds 4096 bytes")
		}
		if err = localconfig.StoreCredential(resolved.CredentialsPath, *url, *project, string(body)); err != nil {
			return err
		}
		printJSON(map[string]any{
			"url": strings.TrimRight(*url, "/"), "project": *project,
			"credentialsFile": resolved.CredentialsPath, "notice": "Project credential stored locally with mode 0600.",
		})
		return nil
	default:
		return errors.New("usage: clank auth set|status")
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
	if isHelp(args[0]) {
		runUsage()
		return nil
	}
	if len(args) > 1 && isHelp(args[1]) {
		runUsage()
		return nil
	}
	if args[0] == "end" {
		f := flag.NewFlagSet("run end", flag.ContinueOnError)
		id := value("CLANKSPACE_RUN", "")
		f.StringVar(&id, "id", id, "run ID")
		f.StringVar(&id, "run", id, "run ID (alias for --id)")
		outcome := f.String("outcome", "completed", "outcome")
		verification := f.String("verification", "", "verification summary")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		o, e := c.EndRun(ctx, id, domain.EndRunInput{Outcome: *outcome, Verification: *verification})
		if e == nil {
			printJSON(o)
		}
		return e
	}
	if args[0] != "start" {
		return errors.New("usage: clank run start|end")
	}
	f := flag.NewFlagSet("run start", flag.ContinueOnError)
	project := f.String("project", os.Getenv("CLANKSPACE_PROJECT"), "project ID or slug")
	agent := f.String("agent", value("CLANKSPACE_AGENT", "agent"), "agent name")
	harness := f.String("harness", value("CLANKSPACE_HARNESS", ""), "harness")
	harnessVersion := f.String("harness-version", value("CLANKSPACE_HARNESS_VERSION", ""), "harness version")
	provider := f.String("provider", value("CLANKSPACE_PROVIDER", ""), "provider")
	model := f.String("model", value("CLANKSPACE_MODEL", ""), "model")
	reasoning := f.String("reasoning", value("CLANKSPACE_REASONING", ""), "reasoning effort")
	role := f.String("role", value("CLANKSPACE_ROLE", "primary"), "primary|subagent|reviewer|automation|integration")
	runType := f.String("type", value("CLANKSPACE_RUN_TYPE", "interactive"), "interactive|automation")
	objective := f.String("objective", "", "objective")
	branch := f.String("branch", value("CLANKSPACE_BRANCH", ""), "git branch")
	worktree := f.String("worktree", value("CLANKSPACE_WORKTREE", ""), "worktree")
	paths := f.String("instructions", value("CLANKSPACE_INSTRUCTIONS", ""), "instruction profile names/hashes, comma separated")
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
		return errors.New("usage: clank note add|create|supersede")
	}
	if isHelp(args[0]) || (len(args) > 1 && isHelp(args[1])) {
		noteUsage()
		return nil
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
	if args[0] != "add" && args[0] != "create" {
		return errors.New("usage: clank note add|create|supersede")
	}
	f := flag.NewFlagSet("note add", flag.ContinueOnError)
	project := f.String("project", os.Getenv("CLANKSPACE_PROJECT"), "project")
	runID := f.String("run", os.Getenv("CLANKSPACE_RUN"), "run ID")
	kind := f.String("kind", "intent", "intent|decision|understanding|observation|checkpoint")
	title := f.String("title", "", "title")
	summary := f.String("summary", "", "concise project implication")
	rationale := f.String("rationale", "", "reasoning summary, not chain-of-thought")
	ledBy := f.String("led-by", "agent", "human|agent|joint|external")
	basis := f.String("basis", "autonomous_agent_judgment", "explicit_human_direction|interpreted_human_intent|joint_reasoning|autonomous_agent_judgment|external_evidence")
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
	if len(args) > 0 && (isHelp(args[0]) || (len(args) > 1 && isHelp(args[1]))) {
		trajectoryUsage()
		return nil
	}
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
func localContext(resolved localconfig.Resolved) error {
	cwd, _ := os.Getwd()
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
		"url": resolved.URL, "project": resolved.Project, "projectFile": resolved.ProjectFilePath,
		"repository": git("remote", "get-url", "origin"),
		"branch":     git("branch", "--show-current"), "head": git("rev-parse", "HEAD"), "worktree": cwd,
		"tokenConfigured": resolved.Token != "", "tokenSource": resolved.TokenSource, "notice": domain.AdvisoryNotice,
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
func isHelp(value string) bool { return value == "help" || value == "--help" || value == "-h" }
func usage() {
	fmt.Fprintln(os.Stdout, `clank context
clank run --help
clank brief --run <id> --objective <text> --paths <comma-separated>
clank why <topic-or-path> --run <id>
clank trajectory start --run <id> --objective <text> --rationale <text> --paths <comma-separated>
clank note add|create|supersede
clank run end --id <id> --outcome <completed|aborted> --verification <text>
clank auth | project | repo | mcp | serve | version`)
}

func runUsage() {
	fmt.Fprintln(os.Stdout, `clank run start [options]
  --project <slug>             defaults to the resolved project
  --agent <name>               agent identity
  --harness <name>             codex, shuv2code, or the actual harness
  --harness-version <version>  when known
  --provider <name>            model provider
  --model <model>              exact model ID
  --reasoning <tier>           none|low|medium|high|xhigh|max|unknown
  --role <role>                primary|subagent|reviewer|automation|integration
  --type <type>                interactive|automation
  --objective <text>           current material task
  --branch <branch>            current branch
  --worktree <path>            current worktree
  --instructions <profiles>    comma-separated instruction names or hashes

The command returns a JSON run object. Pass its top-level id to brief, trajectory, note, and run end.

clank run end --id <run-id> --outcome <completed|aborted> --verification <text>
  --run is accepted as an alias for --id`)
}

func noteUsage() {
	fmt.Fprintln(os.Stdout, `clank note add [options]
  --run <run-id>
  --kind <intent|decision|understanding|observation|checkpoint>
  --title <concise title>
  --summary <project implication>
  --rationale <reasoning summary>
  --led-by <human|agent|joint|external>
  --basis <basis>              explicit_human_direction|interpreted_human_intent|joint_reasoning|autonomous_agent_judgment|external_evidence
  --paths <comma-separated paths>

Common provenance pairs:
  human + explicit_human_direction
  joint + joint_reasoning
  agent + interpreted_human_intent|autonomous_agent_judgment
  external + external_evidence

"create" is accepted as an alias for "add".
clank note supersede --id <note-id> --revision <n> --reason <text>`)
}

func trajectoryUsage() {
	fmt.Fprintln(os.Stdout, `clank trajectory start --run <run-id> --objective <text> --rationale <text> --paths <comma-separated> [--branch <branch>]

Ending the associated run automatically closes its active trajectories.`)
}
