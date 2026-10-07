package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/prasenjeet-symon/ogcode/internal/agent"
	"github.com/prasenjeet-symon/ogcode/internal/bus"
	"github.com/prasenjeet-symon/ogcode/internal/config"
	"github.com/prasenjeet-symon/ogcode/internal/db"
	"github.com/prasenjeet-symon/ogcode/internal/docindex"
	"github.com/prasenjeet-symon/ogcode/internal/mcp"
	"github.com/prasenjeet-symon/ogcode/internal/modelcatalog"
	"github.com/prasenjeet-symon/ogcode/internal/permission"
	"github.com/prasenjeet-symon/ogcode/internal/provider"
	"github.com/prasenjeet-symon/ogcode/internal/question"
	"github.com/prasenjeet-symon/ogcode/internal/search"
	"github.com/prasenjeet-symon/ogcode/internal/server"
	"github.com/prasenjeet-symon/ogcode/internal/session"
	"github.com/prasenjeet-symon/ogcode/internal/skill"
	"github.com/prasenjeet-symon/ogcode/internal/tool"
	"github.com/prasenjeet-symon/ogcode/internal/tui"
	"github.com/prasenjeet-symon/ogcode/internal/usage"
	"github.com/spf13/cobra"
)

var tuiModel string
var tuiContinue bool

// newTUICmd builds the `tui` command. It is a constructor rather than a
// package-level value so the same command can also stand alone as the root of
// the `ogcode-tui` binary (see ExecuteTUI), where no `ogcode` root exists.
func newTUICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Start the interactive terminal UI",
		Long: `Start the interactive terminal UI.

The TUI runs an agent session in-process: the same providers, tools, skills and
permission model the server uses, drawn as a full-screen terminal chat.`,
		RunE: runTUI,
	}
	cmd.Flags().StringVar(&tuiModel, "model", "", "Model ID override (e.g. claude-sonnet-4-5)")
	cmd.Flags().BoolVarP(&tuiContinue, "continue", "c", false, "Resume the most recent session for this directory instead of starting a new one")
	return cmd
}

var tuiCmd = newTUICmd()

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	if path := config.EnsureProjectFile(dir); path != "" {
		slog.Info("created project config file", "path", path)
	}

	// Open project DB
	dbPath := filepath.Join(dir, ".ogcode", "ogcode.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	// Open global config DB (provider keys stored here when set via UI)
	home, _ := os.UserHomeDir()
	globalDBPath := filepath.Join(home, ".ogcode", "config.db")
	if err := os.MkdirAll(filepath.Dir(globalDBPath), 0o755); err != nil {
		return fmt.Errorf("create global config dir: %w", err)
	}
	globalDatabase, err := db.Open(globalDBPath)
	if err != nil {
		return fmt.Errorf("open global config database: %w", err)
	}

	// Match the server: notes off means no notes text reaches an agent prompt.
	installID, _ := server.EnsureInstallID(home)
	notesEnabled := server.NotesEnabled(installID, server.NotesFlagCLITimeout)

	// Build provider registry — env vars take precedence over DB-stored keys
	dbProviderCfgs, _ := session.GetAllProviderConfigs(globalDatabase)
	dbProviderMap := make(map[string]*session.ProviderConfig)
	for _, c := range dbProviderCfgs {
		dbProviderMap[c.ProviderID] = c
	}
	resolveKey := func(envKey, providerID string) string {
		if envKey != "" {
			return envKey
		}
		if c, ok := dbProviderMap[providerID]; ok {
			return c.APIKey
		}
		return ""
	}
	resolveBaseURL := func(envURL, providerID string) string {
		if envURL != "" {
			return envURL
		}
		if c, ok := dbProviderMap[providerID]; ok {
			return c.BaseURL
		}
		return ""
	}

	registry := provider.NewRegistry()
	if key := resolveKey(os.Getenv("ANTHROPIC_API_KEY"), "anthropic"); key != "" {
		baseURL := resolveBaseURL(os.Getenv("ANTHROPIC_BASE_URL"), "anthropic")
		if p, e := provider.NewProviderWithConfig("anthropic", key, baseURL); e == nil {
			registry.Register(p)
		}
	}
	if key := resolveKey(os.Getenv("OPENAI_API_KEY"), "openai"); key != "" {
		baseURL := resolveBaseURL(os.Getenv("OPENAI_BASE_URL"), "openai")
		if p, e := provider.NewProviderWithConfig("openai", key, baseURL); e == nil {
			registry.Register(p)
		}
	}
	if key := resolveKey(os.Getenv("OPENROUTER_API_KEY"), "openrouter"); key != "" {
		if p, e := provider.NewProviderWithConfig("openrouter", key, ""); e == nil {
			registry.Register(p)
		}
	}
	ollamaKey := resolveKey(os.Getenv("OLLAMA_API_KEY"), "ollama")
	ollamaBaseURL := resolveBaseURL(os.Getenv("OLLAMA_BASE_URL"), "ollama")
	if os.Getenv("OLLAMA_BASE_URL") == "" {
		ollamaBaseURL = provider.PreferLiveOllamaEndpoint(ollamaBaseURL, provider.DetectOllama())
	}
	if ollamaKey != "" || ollamaBaseURL != "" {
		if p, e := provider.NewProviderWithConfig("ollama", ollamaKey, ollamaBaseURL); e == nil {
			registry.Register(p)
		}
	}
	// A connected OG Lab subscription runs inference through its gateway.
	if acct, e := session.GetOGXAccount(globalDatabase); e == nil && acct.HasPlan() {
		if p, e := provider.NewOGXProvider(acct.Token); e == nil {
			registry.Register(p)
		}
	}

	if e := modelcatalog.Seed(registry, globalDatabase); e != nil {
		slog.Warn("seed model catalog failed", "err", e)
	}

	defaultProvider := registry.DefaultUsable()
	if defaultProvider == nil {
		return fmt.Errorf("no provider configured — set ANTHROPIC_API_KEY, OPENAI_API_KEY, OPENROUTER_API_KEY, or OLLAMA_BASE_URL")
	}

	// Tool registry: the same core set the server and one-shot run use.
	docindexStore := docindex.NewStore(database)
	toolRegistry := tool.NewRegistry()
	tool.RegisterCoreTools(toolRegistry, docindexStore)

	fullCfg := config.Load(dir)
	skillCfg := fullCfg.Skills
	skillLoader := skill.NewLoader(skill.Config{
		Paths:       skillCfg.Paths,
		URLs:        skillCfg.URLs,
		Permissions: skillCfg.Permissions,
	})
	toolRegistry.Register(tool.NewSkillTool(skillLoader))

	// MCP servers: connect eagerly so the agent's first step has the tools.
	mcpMgr, mcpErr := mcp.New(context.Background(), fullCfg)
	if mcpErr != nil {
		slog.Warn("mcp: manager construction failed", "err", mcpErr)
	}
	mcpTools, connErr := mcpMgr.Connect(context.Background())
	if connErr != nil {
		slog.Warn("mcp: one or more servers failed to connect", "err", connErr)
	}
	for _, t := range mcpTools {
		toolRegistry.Register(t)
	}
	defer mcpMgr.Close()

	b := bus.New(1024)
	store := session.NewStore(database)

	// Web search, on the same terms as the server.
	searchCfg, err := session.GetSearchConfig(globalDatabase)
	if err != nil {
		slog.Warn("failed to read search config from DB; leaving web search enabled on the native engine", "err", err)
		searchCfg = &session.SearchConfig{Enabled: true, Provider: session.SearchProviderNative}
	}
	searchEnabled := searchCfg.Enabled
	if v := os.Getenv("OGCODE_SEARCH_ENABLED"); v != "" {
		searchEnabled = strings.EqualFold(v, "true")
	}
	var searchBridge search.Backend
	if searchEnabled {
		searchBridge = search.BuildBackend(searchCfg.Provider, searchCfg.TavilyAPIKey)
		toolRegistry.Register(tool.WebSearchTool{Bridge: searchBridge})
		toolRegistry.Register(tool.FetchPageTool{Bridge: searchBridge})
	} else {
		slog.Info("web search disabled by configuration")
	}

	ledger := usage.NewLedger(globalDatabase, dir)
	if _, _, err := ledger.Backfill(store); err != nil {
		slog.Warn("usage ledger backfill", "err", err)
	}
	ledger.SetHosts(registry.EndpointHost)

	permissions := permission.NewManager(permission.NewStore(globalDatabase))
	questions := question.NewManager()

	lr := &agent.LoopRunner{
		Store:           store,
		Usage:           ledger,
		Bus:             b,
		Registry:        registry,
		DefaultProvider: defaultProvider,
		Tools:           toolRegistry,
		Dir:             dir,
		MaxSteps:        10000,
		MaxAutoResumes:  2,
		Skills:          skillLoader,
		SearchBridge:    searchBridge,
		Permissions:     permissions,
		Questions:       questions,
		IndexedFileCount: func(dir string) int {
			paths, err := docindexStore.ListDocPaths(dir)
			if err != nil {
				slog.Warn("index status lookup failed, omitting from prompt", "dir", dir, "err", err)
				return -1
			}
			return len(paths)
		},
	}
	lr.NotesEnabled = new(atomic.Bool)
	lr.NotesEnabled.Store(notesEnabled)

	// Tools that close over the runner, now that it exists.
	toolRegistry.Register(tool.TaskTool{Run: lr.RunTaskSession})
	toolRegistry.Register(tool.AskUserTool{Ask: lr.AskUser})
	if searchBridge != nil {
		toolRegistry.Register(tool.DeepSearchTool{Run: lr.RunSearchSession})
	}

	sess, err := resumeOrCreateSession(store, dir, b, permissions)
	if err != nil {
		return err
	}

	app := tui.NewApp(store, b, permissions, questions, registry, lr, dir, sess)
	return app.Run()
}

// resumeOrCreateSession picks the session the TUI should open. With --continue
// it reopens the most recent session for the directory (the same non-note/index
// sessions the server lists); otherwise, or when none exists, it creates a fresh
// one. A --model override is persisted onto whichever session it opens.
func resumeOrCreateSession(store *session.Store, dir string, b *bus.Bus, permissions *permission.Manager) (*session.Session, error) {
	if tuiContinue {
		if sessions, err := store.List(dir); err == nil && len(sessions) > 0 {
			sess := sessions[0]
			if tuiModel != "" && sess.Model != tuiModel {
				sess.Model = tuiModel
				sess.UpdatedAt = session.Now()
				if err := store.Update(sess); err != nil {
					return nil, fmt.Errorf("update session model: %w", err)
				}
			}
			return sess, nil
		}
	}

	sess := &session.Session{
		ID:          session.NewSessionID(),
		ProjectID:   dir,
		Directory:   dir,
		Title:       "New session",
		Model:       tuiModel,
		SessionType: "build",
		Permission:  permissions.DefaultMode(),
		CreatedAt:   session.Now(),
		UpdatedAt:   session.Now(),
	}
	if err := store.Create(sess); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	b.Publish("session.created", sess)
	return sess, nil
}
