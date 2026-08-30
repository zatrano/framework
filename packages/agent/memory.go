package agent

import (
	"sync"

	"github.com/zatrano/framework/packages/ai"
)

// BufferMemory keeps an in-process message list (optionally capped).
type BufferMemory struct {
	mu      sync.Mutex
	msgs    []ai.Message
	MaxKeep int // 0 = unlimited; when exceeded, drops oldest non-system messages
}

// Messages implements Memory.
func (m *BufferMemory) Messages() []ai.Message {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ai.Message(nil), m.msgs...)
}

// Append implements Memory.
func (m *BufferMemory) Append(msgs ...ai.Message) {
	if m == nil || len(msgs) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msgs...)
	m.trimLocked()
}

// Clear implements Memory.
func (m *BufferMemory) Clear() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = nil
}

func (m *BufferMemory) trimLocked() {
	if m.MaxKeep <= 0 || len(m.msgs) <= m.MaxKeep {
		return
	}
	// Preserve leading system messages.
	sys := 0
	for sys < len(m.msgs) && m.msgs[sys].Role == "system" {
		sys++
	}
	drop := len(m.msgs) - m.MaxKeep
	if drop <= 0 {
		return
	}
	start := sys + drop
	if start < sys {
		start = sys
	}
	if start > len(m.msgs) {
		start = len(m.msgs)
	}
	kept := append([]ai.Message{}, m.msgs[:sys]...)
	kept = append(kept, m.msgs[start:]...)
	m.msgs = kept
}
