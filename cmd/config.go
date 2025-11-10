package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ionut-t/perp/internal/config"
	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.New()
			if err != nil {
				fmt.Println("Error loading config:", err)
				return
			}

			configPath := config.GetConfigFilePath()

			editorFlag, _ := cmd.Flags().GetString(config.EditorKey)
			llmProviderFlag, _ := cmd.Flags().GetString(config.LLMProviderKey)
			llmModelFlag, _ := cmd.Flags().GetString(config.LLMModelKey)

			flagsSet := false

			if editorFlag != "" {
				if err := cfg.SetEditor(editorFlag); err != nil {
					fmt.Println("Error setting editor:", err)
					return
				}
				flagsSet = true
				fmt.Println("Editor set to:", editorFlag)
			}

			if llmProviderFlag != "" {
				if err := cfg.SetLLMProvider(llmProviderFlag); err != nil {
					fmt.Println("Error setting LLM provider:", err)
					return
				}
				flagsSet = true
				fmt.Println("LLM provider set to:", llmProviderFlag)
			}

			if llmModelFlag != "" {
				if err := cfg.SetLLMModel(llmModelFlag); err != nil {
					fmt.Println("Error setting LLM model:", err)
					return
				}
				flagsSet = true
				fmt.Println("LLM model set to:", llmModelFlag)
			}

			if !flagsSet {
				openInEditor(configPath)
			}
		},
	}

	cmd.Flags().StringP(config.EditorKey, "e", "", "Set the editor to use for editing config")
	cmd.Flags().StringP(config.LLMProviderKey, "p", "", "Set the LLM provider (e.g., gemini, vertexai)")
	cmd.Flags().StringP(config.LLMModelKey, "m", "", "Set the LLM model")

	return cmd
}

func openInEditor(configPath string) {
	editor := config.GetEditor()

	cmd := exec.Command(editor, configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Error opening editor:", err)
		os.Exit(1)
	}
}

func initConfig() {
	if _, err := config.InitialiseConfigFile(); err != nil {
		fmt.Printf("Error initializing config: %v\n", err)
	}
}
