package agent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// ThinkingRecorder persists AI reasoning content to file and/or log.
// It is used during the ReAct loop to capture thinking/reasoning chunks
// produced by the AI model.
type ThinkingRecorder struct {
	sessionID uint
	filePath  string
	mu        sync.Mutex
	file      *os.File   // nil when file recording is disabled
	logger    *log.Logger // nil when log recording is disabled
}

// NewThinkingRecorder creates a ThinkingRecorder for the given session.
//   - If AGENT_THINKING_FILE=true, writes to .thinking/{session_id}.md
//   - If AGENT_THINKING_LOG=true, logs each chunk with prefix [agent:thinking]
func NewThinkingRecorder(sessionID uint) *ThinkingRecorder {
	r := &ThinkingRecorder{
		sessionID: sessionID,
	}

	if os.Getenv("AGENT_THINKING_FILE") == "true" {
		dir := ".thinking"
		if err := os.MkdirAll(dir, 0755); err == nil {
			path := filepath.Join(dir, fmt.Sprintf("%d.md", sessionID))
			f, err := os.Create(path)
			if err == nil {
				r.file = f
				r.filePath = path
			}
		}
	}

	if os.Getenv("AGENT_THINKING_LOG") == "true" {
		r.logger = log.New(os.Stderr, "[agent:thinking] ", log.Ltime|log.Lmicroseconds)
	}

	return r
}

// Write records a thinking chunk to the configured outputs (file, log).
// Thread-safe.
func (r *ThinkingRecorder) Write(chunk string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file != nil {
		_, _ = r.file.WriteString(chunk)
	}
	if r.logger != nil {
		r.logger.Print(chunk)
	}
}

// Close closes the underlying file handle. Safe to call multiple times.
func (r *ThinkingRecorder) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file != nil {
		_ = r.file.Close()
		r.file = nil
	}
}
