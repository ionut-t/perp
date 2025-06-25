package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ionut-t/perp/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Run: func(cmd *cobra.Command, args []string) {
			configPath := config.GetConfigFilePath()

			editorFlag, _ := cmd.Flags().GetString(config.EditorKey)
			llmProviderFlag, _ := cmd.Flags().GetString(config.LLMProviderKey)
			llmApiKeyFlag, _ := cmd.Flags().GetString(config.LLMApiKey)
			llmModelFlag, _ := cmd.Flags().GetString(config.LLMModelKey)
			vertexAILocationFlag, _ := cmd.Flags().GetString(config.VertexAILocationKey)
			vertexAIProjectIDFlag, _ := cmd.Flags().GetString(config.VertexAIProjectIDKey)

			flagsSet := false

			if editorFlag != "" {
				viper.Set(config.EditorKey, editorFlag)
				flagsSet = true
				fmt.Println("Editor set to:", editorFlag)
			}

			if llmProviderFlag != "" {
				viper.Set(config.LLMProviderKey, llmProviderFlag)
				flagsSet = true
				fmt.Println("LLM provider set to:", llmProviderFlag)
			}

			if llmApiKeyFlag != "" {
				viper.Set(config.LLMApiKey, llmApiKeyFlag)
				flagsSet = true
				fmt.Println("Gemini API key set")
			}

			if llmModelFlag != "" {
				viper.Set(config.LLMModelKey, llmModelFlag)
				flagsSet = true
				fmt.Println("LLM model set to:", llmModelFlag)
			}

			if vertexAIProjectIDFlag != "" {
				viper.Set(config.VertexAIProjectIDKey, vertexAIProjectIDFlag)
				flagsSet = true
				fmt.Println("Vertex AI project ID set to:", vertexAIProjectIDFlag)
			}

			if vertexAILocationFlag != "" {
				viper.Set(config.VertexAILocationKey, vertexAILocationFlag)
				flagsSet = true
				fmt.Println("Vertex AI location set to:", vertexAILocationFlag)
			}

			if flagsSet {
				if err := viper.WriteConfig(); err != nil {
					fmt.Println("Error writing config:", err)
					os.Exit(1)
				}
			}

			if !flagsSet {
				openInEditor(configPath)
			}
		},
	}

	cmd.Flags().StringP(config.EditorKey, "e", "", "Set the editor to use for editing config")
	cmd.Flags().StringP(config.LLMProviderKey, "p", "", "Set the LLM provider (e.g., gemini, vertexai)")
	cmd.Flags().StringP(config.LLMApiKey, "k", "", "Set the LLM API key")
	cmd.Flags().StringP(config.LLMModelKey, "m", "", "Set the LLM model")
	cmd.Flags().StringP(config.VertexAILocationKey, "l", "", "Set the Vertex AI location")
	cmd.Flags().StringP(config.VertexAIProjectIDKey, "v", "", "Set the Vertex AI project ID")

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
