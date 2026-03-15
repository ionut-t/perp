package debug

import (
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
)

func isDebug() bool {
	return len(os.Getenv("DEBUG")) > 0
}

func Listen() (func(), error) {
	if isDebug() {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			return nil, err
		}
		return func() {
			if err := f.Close(); err != nil {
				log.Printf("Error closing debug log file: %v", err)
			}
		}, nil
	}

	return func() {}, nil
}

func Println(args ...any) {
	if isDebug() {
		log.Println(args...)
	}
}

func Printf(format string, args ...any) {
	if isDebug() {
		log.Printf(format, args...)
	}
}
