package cli

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/0to1a/sweo/internal/config"
	"github.com/0to1a/sweo/internal/engine"
	"github.com/0to1a/sweo/internal/server"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start [project]",
		Short: "Start the orchestrator and web dashboard",
		RunE:  runStart,
	}
}

func runStart(cmd *cobra.Command, args []string) error {
	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Println(bold("sweo — SWE-Orchestrator"))
	fmt.Println()

	// Load config
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Preflight checks
	if !preflight(cfg) {
		return fmt.Errorf("preflight checks failed, run 'sweo doctor' for details")
	}

	// Create session manager and lifecycle engine
	sm := engine.NewSessionManager(cfg)
	lc := engine.NewLifecycle(sm, cfg)

	// Start HTTP server
	srv := server.New(cfg, sm)
	go func() {
		log.Printf("Dashboard: %s", cyan(fmt.Sprintf("http://localhost:%d", cfg.Port)))
		if err := srv.Start(); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Start lifecycle polling
	lc.Start()

	fmt.Printf("\n  Dashboard:  %s\n", cyan(fmt.Sprintf("http://localhost:%d", cfg.Port)))
	fmt.Printf("  Projects:   %d\n", len(cfg.Projects))
	for name, proj := range cfg.Projects {
		fmt.Printf("    • %s (%s, agent: %s)\n", name, proj.Repo, proj.Agent)
	}
	fmt.Println("\nPress Ctrl+C to stop")

	// Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	lc.Stop()
	srv.Stop()
	fmt.Println("Stopped. Tmux sessions are still running.")

	return nil
}

func preflight(cfg *config.Config) bool {
	ok := true

	if !checkBinary("tmux") {
		log.Println("ERROR: tmux not found")
		ok = false
	}
	if !checkBinary("gh") {
		log.Println("ERROR: gh CLI not found")
		ok = false
	}
	if !checkGHAuth() {
		log.Println("ERROR: gh not authenticated")
		ok = false
	}
	if !checkBinary("git") {
		log.Println("ERROR: git not found")
		ok = false
	}

	for name, proj := range cfg.Projects {
		switch proj.Agent {
		case "claude-code":
			if !checkBinary("claude") {
				log.Printf("ERROR: claude not found (required by project %s)", name)
				ok = false
			}
		case "codex":
			if !checkBinary("codex") {
				log.Printf("ERROR: codex not found (required by project %s)", name)
				ok = false
			}
		}
	}

	return ok
}
