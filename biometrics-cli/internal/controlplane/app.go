package controlplane

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"biometrics-cli/internal/agents/coder"
	"biometrics-cli/internal/agents/fixer"
	"biometrics-cli/internal/agents/integrator"
	"biometrics-cli/internal/agents/planner"
	"biometrics-cli/internal/agents/reporter"
	"biometrics-cli/internal/agents/reviewer"
	"biometrics-cli/internal/agents/scoper"
	"biometrics-cli/internal/agents/tester"
	httpapi "biometrics-cli/internal/api/http"
	"biometrics-cli/internal/auth/codexbroker"
	"biometrics-cli/internal/blueprints"
	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/executor/opencode"
	llmpolicy "biometrics-cli/internal/llm/policy"
	codexprovider "biometrics-cli/internal/llm/providers/codex"
	geminiprovider "biometrics-cli/internal/llm/providers/gemini"
	nimprovider "biometrics-cli/internal/llm/providers/nim"
	llmrouter "biometrics-cli/internal/llm/router"
	"biometrics-cli/internal/policy"
	"biometrics-cli/internal/runtime/actor"
	"biometrics-cli/internal/runtime/background"
	"biometrics-cli/internal/runtime/bus"
	"biometrics-cli/internal/runtime/scheduler"
	"biometrics-cli/internal/runtime/supervisor"
	"biometrics-cli/internal/skillkit"
	"biometrics-cli/internal/skillops"
	store "biometrics-cli/internal/store/sqlite"
)

const (
	defaultPort     = "59013"
	defaultBindAddr = "127.0.0.1"
)

func Run(ctx context.Context) error {
	workspace := detectWorkspace()
	port := strings.TrimPrefix(strings.TrimSpace(os.Getenv("PORT")), ":")
	if port == "" {
		port = defaultPort
	}
	bindAddr := strings.TrimSpace(os.Getenv("BIOMETRICS_BIND_ADDR"))
	if bindAddr == "" {
		bindAddr = defaultBindAddr
	}

	cfg := Config{
		Workspace: workspace,
		Port:      port,
		BindAddr:  bindAddr,
		DBPath:    filepath.Join(workspace, ".biometrics", "v3.db"),
	}
	return RunWithConfig(ctx, cfg)
}

type Config struct {
	Workspace string
	Port      string
	BindAddr  string
	DBPath    string
}

func RunWithConfig(ctx context.Context, cfg Config) error {
	if strings.TrimSpace(cfg.Workspace) == "" {
		cfg.Workspace = detectWorkspace()
	}
	if strings.TrimSpace(cfg.Port) == "" {
		cfg.Port = defaultPort
	}
	cfg.Port = strings.TrimPrefix(strings.TrimSpace(cfg.Port), ":")
	if strings.TrimSpace(cfg.BindAddr) == "" {
		cfg.BindAddr = defaultBindAddr
	}
	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join(cfg.Workspace, ".biometrics", "v3.db")
	}

	s, err := store.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}
	defer s.Close()

	pol := policy.Default()
	eventBus := bus.NewEventBus(s, bus.WithRedactor(pol.Redact))
	super := supervisor.New(eventBus)
	actorSystem := actor.NewSystem(super)
	execAdapter := opencode.NewAdapter(
		opencode.WithInstallEventHook(func(runID, eventType string, payload map[string]string) {
			event := contracts.Event{
				RunID:   runID,
				Type:    eventType,
				Source:  "executor.opencode",
				Payload: payload,
			}
			if _, publishErr := eventBus.Publish(event); publishErr != nil {
				fmt.Fprintf(os.Stderr, "failed to publish opencode install event: %v\n", publishErr)
			}
		}),
	)
	codexAuthBroker := codexbroker.New()
	modelRouter := llmrouter.New(
		llmpolicy.Default(),
		func(event contracts.Event) {
			if _, publishErr := eventBus.Publish(event); publishErr != nil {
				fmt.Fprintf(os.Stderr, "failed to publish llm router event: %v\n", publishErr)
			}
		},
		pol.Redact,
	)
	modelRouter.Register(codexprovider.New(execAdapter, codexAuthBroker, strings.TrimSpace(os.Getenv("BIOMETRICS_CODEX_MODEL_ID"))))
	modelRouter.Register(geminiprovider.New(execAdapter, strings.TrimSpace(os.Getenv("BIOMETRICS_GEMINI_MODEL_ID"))))
	modelRouter.Register(nimprovider.New(execAdapter, strings.TrimSpace(os.Getenv("BIOMETRICS_NIM_MODEL_ID"))))

	if err := mustRegister(actorSystem, "planner", planner.New()); err != nil {
		return err
	}
	if err := mustRegister(actorSystem, "scoper", scoper.New(cfg.Workspace)); err != nil {
		return err
	}
	if err := mustRegister(actorSystem, "coder", coder.New(modelRouter)); err != nil {
		return err
	}
	if err := mustRegister(actorSystem, "tester", tester.New(cfg.Workspace)); err != nil {
		return err
	}
	if err := mustRegister(actorSystem, "reviewer", reviewer.New()); err != nil {
		return err
	}
	if err := mustRegister(actorSystem, "fixer", fixer.New(modelRouter)); err != nil {
		return err
	}
	if err := mustRegister(actorSystem, "integrator", integrator.New()); err != nil {
		return err
	}
	if err := mustRegister(actorSystem, "reporter", reporter.New()); err != nil {
		return err
	}

	actorSystem.Start(ctx)

	blueprintRegistry, err := blueprints.NewRegistry(cfg.Workspace, "")
	if err != nil {
		return fmt.Errorf("init blueprint registry: %w", err)
	}
	blueprintApplier := blueprints.NewApplier(cfg.Workspace, blueprintRegistry)
	skillManager, err := skillkit.NewManager(skillkit.ManagerOptions{
		Workspace: cfg.Workspace,
		CWD:       cfg.Workspace,
		CodexHome: strings.TrimSpace(os.Getenv("CODEX_HOME")),
	})
	if err != nil {
		return fmt.Errorf("init skill manager: %w", err)
	}
	skillManager.SetOperations(skillops.New(cfg.Workspace, strings.TrimSpace(os.Getenv("CODEX_HOME"))))

	manager := scheduler.NewRunManager(s, actorSystem, eventBus, pol, cfg.Workspace, blueprintRegistry, blueprintApplier)
	manager.SetCodexAuthBroker(codexAuthBroker)
	manager.SetModelCatalogProvider(modelRouter)
	manager.SetSkillManager(skillManager)
	apiServer := httpapi.NewServer(manager, eventBus)
	apiServer.SetBackgroundAgents(background.NewManager(modelRouter, eventBus))

	uiDist := filepath.Join(cfg.Workspace, "biometrics-cli", "web-v3", "dist")

	mux := http.NewServeMux()
	apiHandler := apiServer.Handler()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/health") || r.URL.Path == "/metrics" {
			apiHandler.ServeHTTP(w, r)
			return
		}
		serveUI(w, r, uiDist)
	})

	httpServer := &http.Server{
		Addr:              net.JoinHostPort(cfg.BindAddr, cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	fmt.Printf("BIOMETRICS V3 Control Plane running on http://%s:%s\n", cfg.BindAddr, cfg.Port)
	fmt.Printf("Workspace: %s\n", cfg.Workspace)
	fmt.Printf("Database:  %s\n", cfg.DBPath)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

func mustRegister(system *actor.System, name string, handler actor.Handler) error {
	if err := system.Register(name, 32, handler); err != nil {
		return fmt.Errorf("register actor %s: %w", name, err)
	}
	return nil
}

func detectWorkspace() string {
	if env := os.Getenv("BIOMETRICS_WORKSPACE"); env != "" {
		return env
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if filepath.Base(cwd) == "biometrics-cli" {
		return filepath.Dir(cwd)
	}
	return cwd
}

func serveUI(w http.ResponseWriter, r *http.Request, distRoot string) {
	cleanPath := filepath.Clean(r.URL.Path)
	if cleanPath == "/" {
		serveIndex(w, r, distRoot)
		return
	}
	relPath := strings.TrimPrefix(cleanPath, string(filepath.Separator))

	rootAbs, err := filepath.Abs(distRoot)
	if err != nil {
		serveIndex(w, r, distRoot)
		return
	}
	candidate := filepath.Join(rootAbs, relPath)
	if rel, err := filepath.Rel(rootAbs, candidate); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		serveIndex(w, r, distRoot)
		return
	}
	if fileExists(candidate) {
		http.ServeFile(w, r, candidate)
		return
	}

	serveIndex(w, r, distRoot)
}

func serveIndex(w http.ResponseWriter, r *http.Request, distRoot string) {
	indexFile := filepath.Join(distRoot, "index.html")
	if fileExists(indexFile) {
		http.ServeFile(w, r, indexFile)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("web-v3 UI not found; run npm install && npm run build in biometrics-cli/web-v3"))
}

func fileExists(path string) bool {
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return true
	}
	return false
}
