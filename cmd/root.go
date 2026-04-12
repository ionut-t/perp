package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/fang/v2"
	"github.com/ionut-t/coffee/styles"
	"github.com/ionut-t/perp/internal/config"
	"github.com/ionut-t/perp/internal/debug"
	"github.com/ionut-t/perp/internal/version"
	"github.com/ionut-t/perp/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "perp",
	Short: "perp is a TUI application for interacting with PostgreSQL databases.",
	Run: func(cmd *cobra.Command, args []string) {
		url, err := cmd.Flags().GetString("url")
		if err != nil {
			fmt.Printf("Error parsing URL flag: %v\n", err)
			os.Exit(1)
		}
		appUI(url)
	},
	Version: version.Version(),
}

func Execute() {
	cleanup, err := debug.Listen()
	if err != nil {
		fmt.Printf("Error setting up debug logging: %v\n", err)
		os.Exit(1)
	}

	defer cleanup()

	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(llmInstructionsCmd())

	err = fang.Execute(
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

	rootCmd.Flags().StringP("url", "u", "", "PostgreSQL connection URL (e.g. postgres://user:pass@host:5432/db)")

	if err := config.InitializeLLMInstructions(); err != nil {
		fmt.Printf("Error writing default LLM instructions: %v", err)
	}
}

func appUI(url string) {
	c, err := config.New()
	if err != nil {
		log.Fatalf("Error initializing config: %v", err)
	}

	m := tui.New(c, url)

	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running UI: %v\n", err)
		os.Exit(1)
	}
}
