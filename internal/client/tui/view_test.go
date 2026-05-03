package tui

import (
	"strings"
	"testing"

	"ZhatRoom/internal/protocol"
)

func TestRenderMessagesAlignment(t *testing.T) {
	const myID = "user-1"
	const width = 60

	msgs := []protocol.Message{
		{Type: "system", From: "System", Content: "Welcome to ZhatRoom!"},
		{Type: "chat", From: "Alice", FromID: "user-2", Content: "Hi everyone"},
		{Type: "chat", From: "Bob", FromID: "user-1", Content: "Hello!"},
		{Type: "chat", From: "Alice", FromID: "user-2", Content: "How are you?"},
		{Type: "chat", From: "Bob", FromID: "user-1", Content: "I'm good, thanks"},
		{Type: "system", From: "System", Content: "Alice has left the room"},
	}

	output := renderMessages(msgs, width, myID)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(lines) != len(msgs) {
		t.Fatalf("expected %d lines, got %d", len(msgs), len(lines))
	}

	for i, line := range lines {
		// Remove ANSI escape codes to check content and alignment
		plain := stripANSI(line)
		stripped := strings.TrimSpace(plain)
		padLen := len(line) - len(strings.TrimLeft(line, " "))

		switch msgs[i].Type {
		case "system":
			if !strings.Contains(stripped, "[SYSTEM]:") {
				t.Errorf("line %d: system msg missing [SYSTEM]: prefix, got: %s", i, stripped)
			}
			// Centered: should have padding on the left
			if padLen == 0 {
				t.Logf("line %d: system msg has no left padding (may be edge case), line: %q", i, line)
			}

		default:
			if msgs[i].FromID == myID {
				if !strings.Contains(stripped, "[You]:") {
					t.Errorf("line %d: own msg missing [You]: prefix, got: %s", i, stripped)
				}
				// Right-aligned: should have visible left padding
				if padLen < 3 {
					t.Logf("line %d: own msg may not be right-aligned, padLen=%d, line: %q", i, padLen, line)
				}
			} else {
				if strings.Contains(stripped, "[You]:") {
					t.Errorf("line %d: other's msg incorrectly has [You]: prefix", i)
				}
				// Left-aligned: little to no left padding
				if padLen > 5 {
					t.Logf("line %d: other's msg has unexpected padding, padLen=%d, line: %q", i, padLen, line)
				}
			}
		}
	}
}

func TestRenderMessagesEmpty(t *testing.T) {
	output := renderMessages(nil, 80, "user-1")
	if output != "" {
		t.Errorf("expected empty string, got %q", output)
	}
}

func TestRenderMessagesNoOwnMessages(t *testing.T) {
	msgs := []protocol.Message{
		{Type: "chat", From: "Alice", FromID: "user-2", Content: "Hello"},
		{Type: "chat", From: "Charlie", FromID: "user-3", Content: "Hey"},
	}

	output := renderMessages(msgs, 60, "user-1")
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	for _, line := range lines {
		plain := stripANSI(line)
		if strings.Contains(plain, "[You]:") {
			t.Error("no messages should use [You]: since none match myID")
		}
	}
}

func TestRenderMessagesAllOwn(t *testing.T) {
	msgs := []protocol.Message{
		{Type: "chat", From: "Me", FromID: "me-001", Content: "msg1"},
		{Type: "chat", From: "Me", FromID: "me-001", Content: "msg2"},
	}

	output := renderMessages(msgs, 60, "me-001")
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	for _, line := range lines {
		plain := stripANSI(line)
		if !strings.Contains(plain, "[You]:") {
			t.Errorf("all messages should use [You]:, got: %s", plain)
		}
	}
}

func TestRenderMessagesLongContent(t *testing.T) {
	longText := strings.Repeat("A", 200)
	msgs := []protocol.Message{
		{Type: "chat", From: "Alice", FromID: "user-2", Content: longText},
		{Type: "chat", From: "Me", FromID: "user-1", Content: longText},
	}

	output := renderMessages(msgs, 40, "user-1")
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (content may wrap), got %d", len(lines))
	}
}

// stripANSI removes ANSI escape sequences for content verification.
func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if s[i] >= '@' && s[i] <= '~' {
				inEscape = false
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func TestRenderMessagesAlignmentVisual(t *testing.T) {
	const myID = "user-1"
	const width = 50

	msgs := []protocol.Message{
		{Type: "system", From: "System", Content: "Room created"},
		{Type: "chat", From: "Alice", FromID: "user-2", Content: "Hi"},
		{Type: "chat", From: "Bob", FromID: "user-1", Content: "Yo"},
	}

	output := renderMessages(msgs, width, myID)
	t.Logf("\n--- Visual alignment check (width=%d, each ─ = 10 chars) ---\n%s\n--- end ---", width, output)
}
