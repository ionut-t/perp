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
			println("Config file path:", configPath)

			editorFlag, _ := cmd.Flags().GetString("editor")
			geminiApiKeyFlag, _ := cmd.Flags().GetString("GEMINI_API_KEY")
			useDatabaseSchemaFlag, _ := cmd.Flags().GetBool("use-database-schema")

			flagsSet := false

			if editorFlag != "" {
				viper.Set("editor", editorFlag)
				flagsSet = true
				fmt.Println("Editor set to:", editorFlag)
			}

			if geminiApiKeyFlag != "" {
				viper.Set("GEMINI_API_KEY", geminiApiKeyFlag)
				flagsSet = true
				fmt.Println("Gemini API key set")
			}

			if useDatabaseSchemaFlag {
				viper.Set("use_database_schema", useDatabaseSchemaFlag)
				flagsSet = true
				fmt.Println("use_database_schema set to:", useDatabaseSchemaFlag)
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

	cmd.Flags().StringP("editor", "e", "", "Set the editor to use for editing config")
	cmd.Flags().StringP("gemini-api-key", "g", "", "Set the Gemini API key")
	cmd.Flags().BoolP("use-database-schema", "d", false, "Use database schema for LLM instructions")

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
