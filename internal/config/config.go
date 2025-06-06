package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const rootDir = ".perp"
const configFileName = "config.toml"

type Config interface {
	Editor() string
	Storage() string
	SetEditor(editor string) error
	GetGeminiApiKey() (string, error)
}

type config struct {
	editor  string
	storage string
}

func New() (Config, error) {
	editor := getDefaultEditor()
	storage, err := GetStorage()

	if err != nil {
		return nil, err
	}

	return &config{
		editor:  editor,
		storage: storage,
	}, nil
}

func (c *config) Editor() string {
	return c.editor
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
	return GetGeminiApiKey()
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

func GetGeminiApiKey() (string, error) {
	apiKey := viper.GetString("GEMINI_API_KEY")

	if apiKey == "" {
		return "", errors.New("GEMINI_API_KEY not set")
	}

	return apiKey, nil
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
