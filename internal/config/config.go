package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ionut-t/perp/internal/constants"
	"github.com/spf13/viper"
)

const rootDir = ".perp"
const configFileName = "config.toml"
const llmInstructionsFileName = "llm_instructions.md"

type Config interface {
	Editor() string
	Storage() string
	SetEditor(editor string) error
	GetGeminiApiKey() (string, error)
	GetLLMInstructions() (string, error)
	ShouldUseDatabaseSchema() bool
	EnableLLMDatabaseSchema(enabled bool) error
}

type config struct {
	storage string
}

func New() (Config, error) {
	storage, err := GetStorage()

	if err != nil {
		return nil, err
	}

	return &config{
		storage: storage,
	}, nil
}

func (c *config) Editor() string {
	return GetEditor()
}

func (c *config) Storage() string {
	return c.storage
}

func (m *config) SetEditor(editor string) error {
	if _, err := InitialiseConfigFile(); err != nil {
		return err
	}

	if editor == m.Editor() {
		return nil
	}

	viper.Set("editor", editor)

	return viper.WriteConfig()
}

func (c *config) GetGeminiApiKey() (string, error) {
	apiKey := viper.GetString("GEMINI_API_KEY")

	if apiKey == "" {
		return "", errors.New("GEMINI_API_KEY not set")
	}

	return apiKey, nil
}

func (c *config) GetLLMInstructions() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	llmInstructionsPath := filepath.Join(home, rootDir, llmInstructionsFileName)

	if _, err := os.Stat(llmInstructionsPath); os.IsNotExist(err) {
		defaultInstructions := strings.TrimSpace(constants.LLMDefaultInstructions)
		if err := os.WriteFile(llmInstructionsPath, []byte(defaultInstructions), 0644); err != nil {
			return "", err
		}
	}

	content, err := os.ReadFile(llmInstructionsPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (c *config) ShouldUseDatabaseSchema() bool {
	return viper.GetBool("use_database_schema")
}

func (c *config) EnableLLMDatabaseSchema(enabled bool) error {
	if enabled == c.ShouldUseDatabaseSchema() {
		return nil
	}

	viper.Set("use_database_schema", enabled)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func getDefaultEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	if os.Getenv("WINDIR") != "" {
		return "notepad"
	}

	return "vim"
}

func GetEditor() string {
	editor := viper.GetString("editor")

	if editor == "" {
		return getDefaultEditor()
	}

	return editor
}

func InitialiseConfigFile() (string, error) {
	configPath := viper.ConfigFileUsed()

	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		dir := filepath.Join(home, rootDir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}

		configPath = filepath.Join(dir, configFileName)
		viper.SetConfigFile(configPath)

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			viper.SetDefault("editor", GetEditor())
			viper.SetDefault("use_database_schema", false)

			if err := viper.WriteConfig(); err != nil {
				return "", err
			}

			fmt.Println("Created config at", configPath)
		} else {
			viper.SetConfigFile(configPath)
			_ = viper.ReadInConfig()
		}
	}

	return configPath, nil
}

func GetConfigFilePath() string {
	return viper.ConfigFileUsed()
}

func GetStorage() (string, error) {
	storage := viper.GetString("storage")

	if storage != "" {
		return storage, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, rootDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	return dir, nil
}

func GetLLMInstructionsFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, rootDir, llmInstructionsFileName)
}
