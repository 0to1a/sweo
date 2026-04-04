package cli

import (
	"errors"
	"fmt"

	"github.com/fatih/color"
	"github.com/0to1a/sweo/internal/config"
	"github.com/spf13/cobra"
)

func NewRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sweo",
		Short: "SWE-Orchestrator — manage parallel AI coding agents",
		Long:  "sweo spawns and manages AI coding agents (Claude Code, Codex) across GitHub issues.",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	cmd.AddCommand(newVersionCmd(version))
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newDoctorCmd())

	return cmd
}

// loadConfig loads the config, handling the first-run case where a default config is created.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrConfigCreated) {
			cyan := color.New(color.FgCyan).SprintFunc()
			yellow := color.New(color.FgYellow).SprintFunc()
			configPath := config.ConfigDir() + "/config.yaml"

			fmt.Println(yellow("Welcome to sweo!"))
			fmt.Println()
			fmt.Printf("  A default config has been created at:\n")
			fmt.Printf("  %s\n\n", cyan(configPath))
			fmt.Println("  Edit it to add your project(s), then run sweo again.")
			fmt.Println()
			fmt.Println("  Example project config:")
			fmt.Println()
			fmt.Printf("    %s\n", cyan("projects:"))
			fmt.Printf("    %s\n", cyan("  my-project:"))
			fmt.Printf("    %s\n", cyan("    repo: \"org/repo\""))
			fmt.Printf("    %s\n", cyan("    path: \"/home/you/code/repo\""))
			fmt.Println()
			return nil, fmt.Errorf("please configure %s first", configPath)
		}
		return nil, err
	}
	return cfg, nil
}

func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print sweo version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("sweo", version)
		},
	}
}
