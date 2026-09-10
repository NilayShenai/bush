package builtins

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type ShellContext struct {
	Stdin          io.Reader
	Stdout         io.Writer
	Stderr         io.Writer
	LastExitCode   int
	LastCommand    string
	StartTime      int64
	CommandCount   int
	CommandFreq    map[string]int
	Aliases        map[string]string
	Bookmarks      map[string]string
	HistoryList    []string
	SubshellRunner func(string) (string, error)
	ExitShell      func(code int)
	mu             sync.Mutex
}

type BuiltinFunc func(args []string, ctx *ShellContext) int

var (
	builtins   = make(map[string]BuiltinFunc)
	registryMu sync.RWMutex
)

func Register(name string, fn BuiltinFunc) {
	registryMu.Lock()
	defer registryMu.Unlock()
	builtins[name] = fn
}

func IsBuiltin(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, ok := builtins[name]
	return ok
}

func GetBuiltin(name string) (BuiltinFunc, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	fn, ok := builtins[name]
	return fn, ok
}

func GetBuiltinNames() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	var names []string
	for name := range builtins {
		names = append(names, name)
	}
	return names
}

func bookmarksPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".bush_marks"
	}
	return filepath.Join(home, ".bush_marks")
}

func LoadBookmarks() map[string]string {
	bm := make(map[string]string)
	data, err := os.ReadFile(bookmarksPath())
	if err != nil {
		return bm
	}
	_ = json.Unmarshal(data, &bm)
	return bm
}

func SaveBookmarks(bm map[string]string) error {
	data, err := json.MarshalIndent(bm, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(bookmarksPath(), data, 0600)
}
