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
	EditorKey            = "EDITOR"
	MaxHistoryLengthKey  = "MAX_HISTORY_LENGTH"
	MaxHistoryDaysKey    = "MAX_HISTORY_AGE_IN_DAYS"
	LLMProviderKey       = "LLM_PROVIDER"
	LLMApiKey            = "LLM_API_KEY"
	LLMModelKey          = "LLM_MODEL"
	VertexAIProjectIDKey = "VERTEXAI_PROJECT_ID"
	VertexAILocationKey  = "VERTEXAI_LOCATION"

	rootDir                 = ".perp"
	configFileName          = ".config.toml"
	llmInstructionsFileName = "llm_instructions.md"
	llmInstructions         = "LLM_INSTRUCTIONS"
	llmDefaultInstructions  = "LLM_DEFAULT_INSTRUCTIONS"
)

type Config interface {
	Editor() string
	Storage() string
	SetEditor(editor string) error
	GetMaxHistoryLength() int
	GetMaxHistoryDays() int
	GetLLMProvider() (string, error)
	SetLLMProvider(provider string) error
	GetLLMApiKey() (string, error)
	GetLLMModel() (string, error)
	SetLLMModel(model string) error
	GetVertexAIProjectID() (string, error)
	GetVertexAILocation() (string, error)
	SetVertexAIProjectID(projectID string) error
	SetVertexAILocation(location string) error
	GetLLMInstructions() string
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

func (c *config) GetLLMApiKey() (string, error) {
	apiKey := viper.GetString(LLMApiKey)

	if apiKey == "" {
		return "", fmt.Errorf("%s not set", LLMApiKey)
	}

	return apiKey, nil
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

func (c *config) GetVertexAIProjectID() (string, error) {
	projectID := viper.GetString(VertexAIProjectIDKey)

	if projectID == "" {
		return "", fmt.Errorf("%s not set", VertexAIProjectIDKey)
	}

	return projectID, nil
}

func (c *config) GetVertexAILocation() (string, error) {
	location := viper.GetString(VertexAILocationKey)

	if location == "" {
		return "", fmt.Errorf("%s not set", VertexAILocationKey)
	}

	return location, nil
}

func (c *config) SetVertexAIProjectID(projectID string) error {
	if projectID == "" {
		return fmt.Errorf("%s cannot be empty", VertexAIProjectIDKey)
	}

	viper.Set(VertexAIProjectIDKey, projectID)

	return viper.WriteConfig()
}

func (c *config) SetVertexAILocation(location string) error {
	if location == "" {
		return fmt.Errorf("%s cannot be empty", VertexAILocationKey)
	}

	viper.Set(VertexAILocationKey, location)

	return viper.WriteConfig()
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
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}

		configPath = filepath.Join(dir, configFileName)
		viper.SetConfigFile(configPath)

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			viper.SetDefault(EditorKey, GetEditor())
			viper.SetDefault(MaxHistoryLengthKey, 1000)
			viper.SetDefault(MaxHistoryDaysKey, 90)
			viper.SetDefault(LLMProviderKey, "")
			viper.SetDefault(LLMApiKey, "")
			viper.SetDefault(LLMModelKey, "")
			viper.SetDefault(VertexAIProjectIDKey, "")
			viper.SetDefault(VertexAILocationKey, "")

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
		if err := os.WriteFile(llmInstructionsPath, []byte(defaultLLMInstructions), 0644); err != nil {
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

func writeDefaultConfig() error {
	var sb strings.Builder

	sb.WriteString("# This is the configuration file for perp\n")
	sb.WriteString("\n")

	sb.WriteString("# The keys are case insensitive\n")
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

	sb.WriteString("# The LLM API key is required if the provider is Gemini\n")
	sb.WriteString(fmt.Sprintf("%s = '%s'\n", LLMApiKey, viper.GetString(LLMApiKey)))
	sb.WriteString("\n")

	sb.WriteString("# The LLM model is required for both Gemini and VertexAI\n")
	sb.WriteString("# ex: 'gemini-2.5-pro'\n")
	sb.WriteString(fmt.Sprintf("%s = '%s'\n", LLMModelKey, viper.GetString(LLMModelKey)))
	sb.WriteString("\n")

	sb.WriteString("# The Vertex AI project ID is required if the provider is VertexAI\n")
	sb.WriteString(fmt.Sprintf("%s = '%s'\n", VertexAIProjectIDKey, viper.GetString(VertexAIProjectIDKey)))
	sb.WriteString("\n")

	sb.WriteString("# The Vertex AI location is required if the provider is VertexAI\n")
	sb.WriteString(fmt.Sprintf("%s = '%s'\n", VertexAILocationKey, viper.GetString(VertexAILocationKey)))

	return os.WriteFile(GetConfigFilePath(), []byte(sb.String()), 0644)
}
