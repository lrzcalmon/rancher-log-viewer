package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logFile *os.File
	mu      sync.Mutex
)

func Init() error {
	exe, err := os.Executable()
	if err != nil {
		exe = "."
	}
	dir := filepath.Dir(exe)
	path := filepath.Join(dir, "debug.log")

	logFile, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Fallback to current directory
		logFile, err = os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open debug.log: %w", err)
		}
	}
	return nil
}

func Log(level, msg string) {
	mu.Lock()
	defer mu.Unlock()
	if logFile == nil {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(logFile, "[%s] [%s] %s\n", timestamp, level, msg)
}

func Info(msg string) {
	Log("INFO", msg)
}

func Error(msg string) {
	Log("ERROR", msg)
}

func Errorf(format string, args ...interface{}) {
	Log("ERROR", fmt.Sprintf(format, args...))
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		logFile.Close()
	}
}
