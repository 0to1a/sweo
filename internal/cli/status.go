package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/0to1a/sweo/internal/core"
	"github.com/0to1a/sweo/internal/engine"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show session status",
		RunE:  runStatus,
	}

	cmd.Flags().BoolP("watch", "w", false, "Refresh every 5 seconds")
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
	watch, _ := cmd.Flags().GetBool("watch")
	jsonOut, _ := cmd.Flags().GetBool("json")

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	sm := engine.NewSessionManager(cfg)

	if watch {
		return watchStatus(sm, jsonOut)
	}

	return printStatus(sm, jsonOut)
}

func printStatus(sm *engine.SessionManager, jsonOut bool) error {
	sessions, err := sm.ListAll()
	if err != nil {
		return err
	}

	if jsonOut {
		data, _ := json.MarshalIndent(sessions, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(sessions) == 0 {
		fmt.Println("No active sessions")
		return nil
	}

	printTable(sessions)
	return nil
}

func watchStatus(sm *engine.SessionManager, jsonOut bool) error {
	for {
		// Clear screen
		fmt.Print("\033[H\033[2J")

		if err := printStatus(sm, jsonOut); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

		fmt.Printf("\nRefreshing every 5s... (Ctrl+C to stop)")
		time.Sleep(5 * time.Second)
	}
}

func printTable(sessions []*core.Session) {
	bold := color.New(color.Bold).SprintFunc()

	// Header
	fmt.Printf("  %s  %s  %s  %s  %s  %s\n",
		bold(pad("SESSION", 15)),
		bold(pad("PROJECT", 12)),
		bold(pad("STATUS", 20)),
		bold(pad("ISSUE", 8)),
		bold(pad("PR", 6)),
		bold(pad("AGE", 8)),
	)
	fmt.Println("  " + strings.Repeat("─", 75))

	for _, s := range sessions {
		age := formatAge(time.Since(s.CreatedAt))
		status := colorStatus(s.Status)
		issue := s.IssueID
		if issue == "" {
			issue = "-"
		}
		pr := "-"
		if s.PRNumber > 0 {
			pr = fmt.Sprintf("#%d", s.PRNumber)
		}

		fmt.Printf("  %s  %s  %s  %s  %s  %s\n",
			pad(s.ID, 15),
			pad(s.ProjectID, 12),
			pad(status, 20),
			pad(issue, 8),
			pad(pr, 6),
			pad(age, 8),
		)
	}

	fmt.Printf("\n  Total: %d sessions\n", len(sessions))
}

func colorStatus(status core.SessionStatus) string {
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	switch status {
	case core.StatusWorking:
		return green(string(status))
	case core.StatusPROpen, core.StatusReviewPending:
		return cyan(string(status))
	case core.StatusCIFailed, core.StatusChangesRequested:
		return yellow(string(status))
	case core.StatusErrored, core.StatusStuck:
		return red(string(status))
	case core.StatusMerged, core.StatusDone:
		return green(string(status))
	default:
		return string(status)
	}
}

func formatAge(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
