package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ionut-t/perp/internal/constants"
	"github.com/spf13/viper"
)

const (
	LLMProviderKey       = "LLM_PROVIDER"
	LLMApiKey            = "LLM_API_KEY"
	LLMModelKey          = "LLM_MODEL"
	EditorKey            = "editor"
	VertexAIProjectIDKey = "VERTEX_AI_PROJECT_ID"
	VertexAILocationKey  = "VERTEX_AI_LOCATION"

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
	GetLLMProvider() (string, error)
	SetLLMProvider(provider string) error
	GetLLMApiKey() (string, error)
	GetLLMModel() (string, error)
	SetLLMModel(model string) error
	GetVertexAIProjectID() (string, error)
	GetVertexAILocation() (string, error)
	SetVertexAIProjectID(projectID string) error
	SetVertexAILocation(location string) error
	GetLLMInstructions() (string, error)
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

func (c *config) GetLLMProvider() (string, error) {
	provider := viper.GetString(LLMProviderKey)

	if provider == "" {
		return "", fmt.Errorf("%s not set", LLMProviderKey)
	}

	return provider, nil
}

func (c *config) SetLLMProvider(provider string) error {
	if _, err := InitialiseConfigFile(); err != nil {
		return err
	}

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
	if _, err := InitialiseConfigFile(); err != nil {
		return err
	}

	if model == "" {
		return fmt.Errorf("%s cannot be empty", LLMModelKey)
	}

	viper.Set(LLMModelKey, model)

	return viper.WriteConfig()
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
	if _, err := InitialiseConfigFile(); err != nil {
		return err
	}

	if projectID == "" {
		return fmt.Errorf("%s cannot be empty", VertexAIProjectIDKey)
	}

	viper.Set(VertexAIProjectIDKey, projectID)

	return viper.WriteConfig()
}

func (c *config) SetVertexAILocation(location string) error {
	if _, err := InitialiseConfigFile(); err != nil {
		return err
	}

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
			viper.SetDefault(LLMApiKey, "")
			viper.SetDefault(LLMModelKey, "")
			viper.SetDefault(VertexAIProjectIDKey, "")
			viper.SetDefault(VertexAILocationKey, "")

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
