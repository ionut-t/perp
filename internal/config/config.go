package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/viper"
)

//go:embed config.toml
var defaultConfig string

//go:embed llm_instructions.md
var defaultLLMInstructions string

const (
	EditorKey           = "editor"
	MaxHistoryLengthKey = "max_history_length"
	MaxHistoryDaysKey   = "max_history_days"
	LLMProviderKey      = "llm_provider"
	LLMModelKey         = "llm_model"
	AutoUpdateKey       = "auto_update"
	LeaderKey           = "leader_key"

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

type configData struct {
	Editor           string
	MaxHistoryLength int
	MaxHistoryDays   int
	LLMProvider      string
	LLMModel         string
	AutoUpdate       bool
	LeaderKey        string
}

type config struct {
	storage string
	data    configData
}

func getConfigData() configData {
	return configData{
		Editor:           GetEditor(),
		MaxHistoryLength: viper.GetInt(MaxHistoryLengthKey),
		MaxHistoryDays:   viper.GetInt(MaxHistoryDaysKey),
		LLMProvider:      viper.GetString(LLMProviderKey),
		LLMModel:         viper.GetString(LLMModelKey),
		AutoUpdate:       viper.GetBool(AutoUpdateKey),
		LeaderKey:        viper.GetString(LeaderKey),
	}
}

func New() (Config, error) {
	storage, err := GetStorage()
	if err != nil {
		return nil, err
	}

	return &config{
		storage: storage,
		data:    getConfigData(),
	}, nil
}

func (c *config) AutoUpdateEnabled() bool {
	return c.data.AutoUpdate
}

func (c *config) GetLeaderKey() string {
	return c.data.LeaderKey
}

func (c *config) SetLeaderKey(key string) error {
	if key == c.GetLeaderKey() {
		return nil
	}

	c.data.LeaderKey = key

	return c.updateValueInConfig(LeaderKey, key)
}

func (c *config) Editor() string {
	return c.data.Editor
}

func (c *config) Storage() string {
	return c.storage
}

func (m *config) SetEditor(editor string) error {
	if editor == m.Editor() {
		return nil
	}

	m.data.Editor = editor

	return m.updateValueInConfig(EditorKey, editor)
}

func (c *config) GetMaxHistoryLength() int {
	return viper.GetInt(MaxHistoryLengthKey)
}

func (c *config) GetMaxHistoryDays() int {
	return viper.GetInt(MaxHistoryDaysKey)
}

func (c *config) GetLLMProvider() (string, error) {
	provider := c.data.LLMProvider

	if provider == "" {
		return "", fmt.Errorf("%s not set", LLMProviderKey)
	}

	return provider, nil
}

func (c *config) SetLLMProvider(provider string) error {
	if provider == "" {
		return fmt.Errorf("%s cannot be empty", LLMProviderKey)
	}

	c.data.LLMProvider = provider

	return c.updateValueInConfig(LLMProviderKey, provider)
}

func (c *config) GetLLMModel() (string, error) {
	model := c.data.LLMModel

	if model == "" {
		return "", fmt.Errorf("%s not set", LLMModelKey)
	}

	return model, nil
}

func (c *config) SetLLMModel(model string) error {
	if model == "" {
		return fmt.Errorf("%s cannot be empty", LLMModelKey)
	}

	c.data.LLMModel = model

	return c.updateValueInConfig(LLMModelKey, model)
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

func (c *config) updateValueInConfig(key, value string) error {
	if _, err := os.Stat(GetConfigFilePath()); os.IsNotExist(err) {
		return writeConfig(c.data)
	}

	content, err := os.ReadFile(GetConfigFilePath())
	if err != nil {
		return err
	}

	lines := bytes.Split(content, []byte("\n"))
	var foundKey bool
	for i, line := range lines {
		if bytes.HasPrefix(bytes.ToLower(line), []byte(key)) {
			lines[i] = fmt.Appendf(nil, "%s = \"%s\"", key, value)
			foundKey = true
			break
		}
	}

	if !foundKey {
		lines = append(lines, fmt.Appendf(nil, "%s = \"%s\"", key, value))
	}

	return os.WriteFile(GetConfigFilePath(), bytes.Join(lines, []byte("\n")), 0o644)
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
			viper.SetDefault(LLMModelKey, "gemini-2.0-flash")
			viper.SetDefault(LeaderKey, " ")

			if err := writeConfig(getConfigData()); err != nil {
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

func writeConfig(config configData) error {
	tmpl, err := template.New("config").Parse(defaultConfig)
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, config); err != nil {
		return err
	}

	return os.WriteFile(GetConfigFilePath(), buf.Bytes(), 0o644)
}
