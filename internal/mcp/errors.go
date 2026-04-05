package mcp

import "errors"

var (
	ErrConnection      = errors.New("connection error")
	ErrProtocol        = errors.New("protocol error")
	ErrTool            = errors.New("tool error")
	ErrServerClosed    = errors.New("server closed")
	ErrEmptyResponse   = errors.New("empty response")
	ErrInvalidResponse = errors.New("invalid response")
)

type ConnectionError struct {
	Op  string
	Err error
}

func (e *ConnectionError) Error() string {
	return "connection " + e.Op + ": " + e.Err.Error()
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

type ProtocolError struct {
	Op      string
	Message string
	Code    int
}

func (e *ProtocolError) Error() string {
	msg := "protocol " + e.Op
	if e.Message != "" {
		msg += ": " + e.Message
	}
	return msg
}

type ToolError struct {
	Name string
	Err  error
}

func (e *ToolError) Error() string {
	return "tool " + e.Name + ": " + e.Err.Error()
}

func (e *ToolError) Unwrap() error {
	return e.Err
}
