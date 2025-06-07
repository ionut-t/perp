package cmd

import (
	"os"
	"os/exec"

	"github.com/ionut-t/perp/internal/config"
	"github.com/spf13/cobra"
)

func llmInstructionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "llm-instructions",
		Short: "Manage LLM instructions",
		Long:  "This command allows you to manage and view LLM instructions.",
		Run: func(cmd *cobra.Command, args []string) {
			openInstructionsInEditor()
		},
	}

	return cmd
}

func openInstructionsInEditor() {
	editor := config.GetEditor()
	configPath := config.GetLLMInstructionsFilePath()

	cmd := exec.Command(editor, configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		println("Error opening LLM instructions in editor:", err.Error())
	}
}
