package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/fang"
	"github.com/ionut-t/perp/internal/config"
	"github.com/ionut-t/perp/internal/version"
	"github.com/ionut-t/perp/tui"
	"github.com/ionut-t/perp/ui/styles"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "perp",
	Short: "perp is a TUI application for interacting with PostgreSQL databases.",
	Run: func(cmd *cobra.Command, args []string) {
		appUI()
	},
	Version: version.Version(),
}

func Execute() {
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(llmInstructionsCmd())

	err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithNotifySignal(os.Interrupt, os.Kill),
		fang.WithColorSchemeFunc(styles.FangColorScheme),
		fang.WithoutCompletions(),
	)

	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate(versionTemplate())

	cobra.OnInitialize(initConfig)

	if err := config.InitializeLLMInstructions(); err != nil {
		fmt.Printf("Error writing default LLM instructions: %v", err)
	}
}

func appUI() {
	c, err := config.New()
	if err != nil {
		log.Fatalf("Error initializing config: %v", err)
	}

	m := tui.New(c)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running UI: %v\n", err)
		os.Exit(1)
	}
}
