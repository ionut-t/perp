package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/perp/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "perp",
	Short: "perp is a TUI application for interacting with PostgreSQL databases.",
	Run: func(cmd *cobra.Command, args []string) {
		chatUI()
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

func chatUI() {
	m := tui.New()

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running UI: %v\n", err)
		os.Exit(1)
	}
}
