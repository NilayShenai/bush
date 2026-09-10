package lineeditor

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type History struct {
	entries []string
	path    string
	mu      sync.RWMutex
}

func NewHistory() *History {
	h := &History{
		path: getHistoryFilePath(),
	}
	h.Load()
	return h
}

func getHistoryFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".bush_history"
	}
	return filepath.Join(home, ".bush_history")
}

func (h *History) Load() {
	h.mu.Lock()
	defer h.mu.Unlock()

	file, err := os.Open(h.path)
	if err != nil {
		return
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	h.entries = lines
}

func (h *History) Add(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == cmd {
		return
	}

	h.entries = append(h.entries, cmd)

	file, err := os.OpenFile(h.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err == nil {
		defer file.Close()
		_, _ = file.WriteString(cmd + "\n")
	}
}

func (h *History) Entries() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	res := make([]string, len(h.entries))
	copy(res, h.entries)
	return res
}

func (h *History) FindSuggestion(prefix string) string {
	if prefix == "" {
		return ""
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for i := len(h.entries) - 1; i >= 0; i-- {
		if strings.HasPrefix(h.entries[i], prefix) && len(h.entries[i]) > len(prefix) {
			return h.entries[i][len(prefix):]
		}
	}
	return ""
}
