package cmd

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/perp/internal/config"
	"github.com/ionut-t/perp/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "perp",
	Short: "perp is a TUI application for interacting with PostgreSQL databases.",
	Run: func(cmd *cobra.Command, args []string) {
		appUI()
	},
}

func Execute() {
	rootCmd.AddCommand(configCmd())
	err := rootCmd.Execute()

	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
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
