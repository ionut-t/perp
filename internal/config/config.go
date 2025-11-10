package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

//go:embed llm_instructions.md
var defaultLLMInstructions string

const (
	EditorKey           = "EDITOR"
	MaxHistoryLengthKey = "MAX_HISTORY_LENGTH"
	MaxHistoryDaysKey   = "MAX_HISTORY_AGE_IN_DAYS"
	LLMProviderKey      = "LLM_PROVIDER"
	LLMModelKey         = "LLM_MODEL"
	AutoUpdateKey       = "AUTO_UPDATE_ENABLED"
	LeaderKey           = "LEADER_KEY"

	rootDir                 = ".perp"
	configFileName          = ".config.toml"
	llmInstructionsFileName = "llm_instructions.md"
)

type Config interface {
	Editor() string
	Storage() string
	SetEditor(editor string) error
	GetMaxHistoryLength() int
	GetMaxHistoryDays() int
	GetLLMProvider() (string, error)
	SetLLMProvider(provider string) error
	GetLLMModel() (string, error)
	SetLLMModel(model string) error
	GetLLMInstructions() string
	AutoUpdateEnabled() bool
	GetLeaderKey() string
	SetLeaderKey(key string) error
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

func (c *config) AutoUpdateEnabled() bool {
	return viper.GetBool(AutoUpdateKey)
}

func (c *config) GetLeaderKey() string {
	return viper.GetString(LeaderKey)
}

func (c *config) SetLeaderKey(key string) error {
	if key == c.GetLeaderKey() {
		return nil
	}

	viper.Set(LeaderKey, key)
	return viper.WriteConfig()
}

func (c *config) Editor() string {
	return GetEditor()
}

func (c *config) Storage() string {
	return c.storage
}

func (m *config) SetEditor(editor string) error {
	if editor == m.Editor() {
		return nil
	}

	viper.Set("editor", editor)

	return viper.WriteConfig()
}

func (c *config) GetMaxHistoryLength() int {
	return viper.GetInt(MaxHistoryLengthKey)
}

func (c *config) GetMaxHistoryDays() int {
	return viper.GetInt(MaxHistoryDaysKey)
}

func (c *config) GetLLMProvider() (string, error) {
	provider := viper.GetString(LLMProviderKey)

	if provider == "" {
		return "", fmt.Errorf("%s not set", LLMProviderKey)
	}

	return provider, nil
}

func (c *config) SetLLMProvider(provider string) error {
	if provider == "" {
		return fmt.Errorf("%s cannot be empty", LLMProviderKey)
	}

	viper.Set(LLMProviderKey, provider)

	return viper.WriteConfig()
}

func (c *config) GetLLMModel() (string, error) {
	model := viper.GetString(LLMModelKey)

	if model == "" {
		return "", fmt.Errorf("%s not set", LLMModelKey)
	}

	return model, nil
}

func (c *config) SetLLMModel(model string) error {
	if model == "" {
		return fmt.Errorf("%s cannot be empty", LLMModelKey)
	}

	viper.Set(LLMModelKey, model)

	return viper.WriteConfig()
}

func (c *config) GetLLMInstructions() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultLLMInstructions
	}

	llmInstructionsPath := filepath.Join(home, rootDir, llmInstructionsFileName)

	content, err := os.ReadFile(llmInstructionsPath)
	if err != nil || len(content) == 0 {
		return defaultLLMInstructions
	}

	return string(content)
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
	editor := viper.GetString(EditorKey)

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
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}

		configPath = filepath.Join(dir, configFileName)
		viper.SetConfigFile(configPath)

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			viper.SetDefault(AutoUpdateKey, true)
			viper.SetDefault(EditorKey, GetEditor())
			viper.SetDefault(MaxHistoryLengthKey, 1000)
			viper.SetDefault(MaxHistoryDaysKey, 90)
			viper.SetDefault(LLMProviderKey, "")
			viper.SetDefault(LLMModelKey, "")
			viper.SetDefault(LeaderKey, " ")

			if err := writeDefaultConfig(); err != nil {
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

func InitializeLLMInstructions() error {
	llmInstructionsPath := GetLLMInstructionsFilePath()

	if _, err := os.Stat(llmInstructionsPath); os.IsNotExist(err) {
		if err := os.WriteFile(llmInstructionsPath, []byte(defaultLLMInstructions), 0o644); err != nil {
			return fmt.Errorf("failed to write LLM instructions: %w", err)
		}
	}

	return nil
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
	if err := os.MkdirAll(dir, 0o755); err != nil {
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

func writeDefaultConfig() error {
	var sb strings.Builder

	sb.WriteString("# This is the configuration file for perp\n")
	sb.WriteString("\n")

	sb.WriteString("# The keys are case insensitive\n")
	sb.WriteString("\n")

	sb.WriteString("# Auto-update feature can be enabled or disabled\n")
	sb.WriteString(fmt.Sprintf("%s = %t\n", AutoUpdateKey, viper.GetBool(AutoUpdateKey)))
	sb.WriteString("\n")

	sb.WriteString("# The editor will be used to edit the config file, LLM instructions and exported data\n")
	sb.WriteString(fmt.Sprintf("%s = '%s'\n", EditorKey, GetEditor()))
	sb.WriteString("\n")

	sb.WriteString("# The maximum number of history entries to keep\n")
	sb.WriteString(fmt.Sprintf("%s = %d\n", MaxHistoryLengthKey, viper.GetInt(MaxHistoryLengthKey)))
	sb.WriteString("\n")

	sb.WriteString("# The maximum number of days to keep history entries\n")
	sb.WriteString(fmt.Sprintf("%s = %d\n", MaxHistoryDaysKey, viper.GetInt(MaxHistoryDaysKey)))
	sb.WriteString("\n")

	sb.WriteString("# It can be set to 'Gemini' or 'VertexAI' (case insensitive)\n")
	sb.WriteString("# If unset, Gemini will be used\n")
	sb.WriteString(fmt.Sprintf("%s = '%s'\n", LLMProviderKey, viper.GetString(LLMProviderKey)))
	sb.WriteString("\n")

	sb.WriteString("# The LLM model is required for both Gemini and VertexAI. Ex: 'gemini-2.5-pro'\n")
	sb.WriteString(fmt.Sprintf("%s = '%s'\n", LLMModelKey, viper.GetString(LLMModelKey)))
	sb.WriteString("\n")

	sb.WriteString("# The leader key used in the TUI. Default is space (' ')\n")
	sb.WriteString(fmt.Sprintf("%s = '%s'\n", LeaderKey, viper.GetString(LeaderKey)))
	sb.WriteString("\n")

	return os.WriteFile(GetConfigFilePath(), []byte(sb.String()), 0o644)
}
