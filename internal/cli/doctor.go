package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check system dependencies and configuration",
		RunE:  runDoctor,
	}
}

func runDoctor(cmd *cobra.Command, args []string) error {
	pass := color.New(color.FgGreen).SprintFunc()
	fail := color.New(color.FgRed).SprintFunc()
	warn := color.New(color.FgYellow).SprintFunc()

	fmt.Println("sweo doctor — checking system health")
	fmt.Println(strings.Repeat("─", 50))

	allOK := true

	// Check tmux
	if checkBinary("tmux") {
		ver := getBinaryVersion("tmux", "-V")
		fmt.Printf("  %s tmux (%s)\n", pass("PASS"), ver)
	} else {
		fmt.Printf("  %s tmux not found\n", fail("FAIL"))
		allOK = false
	}

	// Check git
	if checkBinary("git") {
		ver := getBinaryVersion("git", "--version")
		fmt.Printf("  %s git (%s)\n", pass("PASS"), ver)
	} else {
		fmt.Printf("  %s git not found\n", fail("FAIL"))
		allOK = false
	}

	// Check gh
	if checkBinary("gh") {
		ver := getBinaryVersion("gh", "--version")
		fmt.Printf("  %s gh CLI (%s)\n", pass("PASS"), strings.Split(ver, "\n")[0])
	} else {
		fmt.Printf("  %s gh CLI not found\n", fail("FAIL"))
		allOK = false
	}

	// Check gh auth
	if checkGHAuth() {
		fmt.Printf("  %s gh authenticated\n", pass("PASS"))
	} else {
		fmt.Printf("  %s gh not authenticated (run: gh auth login)\n", fail("FAIL"))
		allOK = false
	}

	// Check config
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("  %s config: %v\n", warn("WARN"), err)
		fmt.Println()
		return nil
	}
	fmt.Printf("  %s config loaded (%d projects)\n", pass("PASS"), len(cfg.Projects))

	// Per-project checks
	for name, proj := range cfg.Projects {
		fmt.Printf("\n  Project: %s (%s)\n", name, proj.Repo)

		// Check path exists
		if _, err := os.Stat(proj.Path); err == nil {
			fmt.Printf("    %s path exists\n", pass("PASS"))
		} else {
			fmt.Printf("    %s path not found: %s\n", fail("FAIL"), proj.Path)
			allOK = false
		}

		// Check agent binary
		switch proj.Agent {
		case "claude-code":
			if checkBinary("claude") {
				fmt.Printf("    %s claude binary\n", pass("PASS"))
			} else {
				fmt.Printf("    %s claude not found\n", fail("FAIL"))
				allOK = false
			}
		case "codex":
			if checkBinary("codex") {
				fmt.Printf("    %s codex binary\n", pass("PASS"))
			} else {
				fmt.Printf("    %s codex not found\n", fail("FAIL"))
				allOK = false
			}
		}

		// Check gh repo access
		if checkGHRepoAccess(proj.Repo) {
			fmt.Printf("    %s repo accessible\n", pass("PASS"))
		} else {
			fmt.Printf("    %s cannot access repo %s\n", warn("WARN"), proj.Repo)
		}
	}

	fmt.Println()
	if allOK {
		fmt.Printf("%s All checks passed\n", pass("✓"))
	} else {
		fmt.Printf("%s Some checks failed\n", fail("✗"))
	}

	return nil
}

func checkBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func getBinaryVersion(name string, arg string) string {
	out, err := exec.Command(name, arg).Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func checkGHAuth() bool {
	cmd := exec.Command("gh", "auth", "status")
	return cmd.Run() == nil
}

func checkGHRepoAccess(repo string) bool {
	cmd := exec.Command("gh", "repo", "view", repo, "--json", "name")
	return cmd.Run() == nil
}
