package history

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ionut-t/perp/internal/config"
)

type HistoryLog struct {
	Query string
	Time  time.Time
}

// Add adds a new query to the history and returns the updated history logs.
func Add(query string) ([]HistoryLog, error) {
	storage, err := config.GetStorage()

	if err != nil {
		return nil, err
	}

	path := filepath.Join(storage, "history")

	var data []byte
	var history []HistoryLog
	if _, err := os.Stat(path); err == nil {
		history, data, err = readHistoryLogs(path, history)

		if err != nil {
			return nil, err
		}
	}

	query = strings.TrimSpace(query)

	if query != "" {
		newLog := HistoryLog{
			Query: query,
			Time:  time.Now(),
		}
		history = append(history, newLog)

		data = append(data, []byte("---\n")...)
		data = append(data, []byte("time: "+newLog.Time.Format(`"2006-01-02T15:04:05Z07:00"`)+"\n")...)
		data = append(data, []byte("query: "+newLog.Query+"\n")...)
		data = append(data, []byte("---\n")...)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write server file: %w", err)
	}

	return getUniqueSortedHistory(history), nil
}

// Get retrieves the history logs from the storage.
func Get() ([]HistoryLog, error) {
	storage, err := config.GetStorage()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(storage, "history")

	var history []HistoryLog
	if _, err := os.Stat(path); err == nil {
		history, _, err = readHistoryLogs(path, history)
		if err != nil {
			return nil, err
		}
	}

	slices.SortFunc(history, func(a, b HistoryLog) int {
		return b.Time.Compare(a.Time)
	})

	return getUniqueSortedHistory(history), nil
}

func readHistoryLogs(path string, history []HistoryLog) ([]HistoryLog, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	entries := bytes.SplitSeq(data, []byte("---"))
	for entry := range entries {
		entry = bytes.TrimSpace(entry)
		if len(entry) == 0 {
			continue
		}
		lines := bytes.Split(entry, []byte("\n"))
		var log HistoryLog
		for _, line := range lines {
			line = bytes.TrimSpace(line)
			if bytes.HasPrefix(line, []byte("time:")) {
				t := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("time:")))
				parsedTime, err := time.Parse(`"2006-01-02T15:04:05Z07:00"`, string(t))
				if err == nil {
					log.Time = parsedTime
				}
			} else if bytes.HasPrefix(line, []byte("query:")) {
				q := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("query:")))
				log.Query = string(bytes.Trim(q, `"`))
			}
		}
		if !log.Time.IsZero() && log.Query != "" {
			history = append(history, log)
		}
	}

	return history, data, nil
}

func getUniqueSortedHistory(history []HistoryLog) []HistoryLog {
	slices.SortFunc(history, func(a, b HistoryLog) int {
		return b.Time.Compare(a.Time)
	})

	uniqueHistory := make([]HistoryLog, 0, len(history))

	seen := make(map[string]bool)
	for _, log := range history {
		query := strings.TrimSpace(log.Query)
		if !seen[query] {
			seen[query] = true
			uniqueHistory = append(uniqueHistory, log)
		}
	}

	return uniqueHistory
}
