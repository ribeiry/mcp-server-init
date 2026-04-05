package mcp

import (
	"errors"
	"testing"
)

func TestConnectionError(t *testing.T) {
	t.Run("Error message", func(t *testing.T) {
		err := &ConnectionError{Op: "connect", Err: errors.New("refused")}
		expected := "connection connect: refused"
		if err.Error() != expected {
			t.Errorf("got %q, want %q", err.Error(), expected)
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		original := errors.New("connection refused")
		err := &ConnectionError{Op: "connect", Err: original}
		unwrapped := err.Unwrap()
		if unwrapped != original {
			t.Errorf("got %v, want %v", unwrapped, original)
		}
	})

	t.Run("errors.Is", func(t *testing.T) {
		err := &ConnectionError{Op: "read", Err: ErrConnection}
		if !errors.Is(err, ErrConnection) {
			t.Error("expected errors.Is to match ErrConnection")
		}
	})
}

func TestProtocolError(t *testing.T) {
	t.Run("Error message with code", func(t *testing.T) {
		err := &ProtocolError{Op: "parse", Message: "invalid json", Code: -32700}
		expected := "protocol parse: invalid json"
		if err.Error() != expected {
			t.Errorf("got %q, want %q", err.Error(), expected)
		}
	})

	t.Run("Error message without message", func(t *testing.T) {
		err := &ProtocolError{Op: "initialize", Code: 0}
		expected := "protocol initialize"
		if err.Error() != expected {
			t.Errorf("got %q, want %q", err.Error(), expected)
		}
	})
}

func TestToolError(t *testing.T) {
	t.Run("Error message", func(t *testing.T) {
		err := &ToolError{Name: "read_file", Err: errors.New("file not found")}
		expected := "tool read_file: file not found"
		if err.Error() != expected {
			t.Errorf("got %q, want %q", err.Error(), expected)
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		original := errors.New("permission denied")
		err := &ToolError{Name: "write_file", Err: original}
		unwrapped := err.Unwrap()
		if unwrapped != original {
			t.Errorf("got %v, want %v", unwrapped, original)
		}
	})

	t.Run("errors.Is with wrapped error", func(t *testing.T) {
		err := &ToolError{Name: "read_file", Err: ErrEmptyResponse}
		if !errors.Is(err, ErrEmptyResponse) {
			t.Error("expected errors.Is to match ErrEmptyResponse")
		}
	})
}

func TestErrorVariables(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrConnection", ErrConnection},
		{"ErrProtocol", ErrProtocol},
		{"ErrTool", ErrTool},
		{"ErrServerClosed", ErrServerClosed},
		{"ErrEmptyResponse", ErrEmptyResponse},
		{"ErrInvalidResponse", ErrInvalidResponse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("expected %s to be non-nil", tt.name)
			}
		})
	}
}
