package clipboard

import (
	"github.com/atotto/clipboard"
)

func Write(text string) error {
	return clipboard.WriteAll(text)
}

func Read() (string, error) {
	text, err := clipboard.ReadAll()

	if err != nil {
		return "", err
	}

	return text, nil
}
