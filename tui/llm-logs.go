package tui

import (
	"time"

	"github.com/ionut-t/perp/ui/list"
)

type chatLog struct {
	Prompt   string
	Response string
	Error    error
	Time     time.Time
}

func processLogs(logs []chatLog) []list.Item {
	items := make([]list.Item, len(logs))

	for i, n := range logs {
		items[i] = list.Item{
			Title:       n.Prompt,
			Subtitle:    n.Time.Format("02/01/2006, 15:04:05"),
			Description: n.Response,
		}
	}

	return items
}
