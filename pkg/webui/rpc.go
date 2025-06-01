package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// RPCRequest represents a JSON-RPC 2.0 request
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// RPCResponse represents a JSON-RPC 2.0 response
type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// RPCError represents a JSON-RPC 2.0 error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// RPCHandler handles JSON-RPC method calls
type RPCHandler struct {
	webui *WebUI
}

// NewRPCHandler creates a new RPC handler
func NewRPCHandler(webui *WebUI) *RPCHandler {
	return &RPCHandler{webui: webui}
}

// HandleRequest processes a JSON-RPC request
func (h *RPCHandler) HandleRequest(ctx context.Context, req *RPCRequest) *RPCResponse {
	response := &RPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "tileset.fetch":
		result, err := h.handleTilesetFetch(ctx, req.Params)
		if err != nil {
			response.Error = h.makeError(InternalError, err.Error())
		} else {
			response.Result = result
		}

	case "game.getState":
		result, err := h.handleGameGetState(ctx, req.Params)
		if err != nil {
			response.Error = h.makeError(InternalError, err.Error())
		} else {
			response.Result = result
		}

	case "game.sendInput":
		result, err := h.handleGameSendInput(ctx, req.Params)
		if err != nil {
			response.Error = h.makeError(InvalidParams, err.Error())
		} else {
			response.Result = result
		}

	case "game.poll":
		result, err := h.handleGamePoll(ctx, req.Params)
		if err != nil {
			response.Error = h.makeError(InternalError, err.Error())
		} else {
			response.Result = result
		}

	case "tileset.update":
		result, err := h.handleTilesetUpdate(ctx, req.Params)
		if err != nil {
			response.Error = h.makeError(InvalidParams, err.Error())
		} else {
			response.Result = result
		}

	case "session.info":
		result, err := h.handleSessionInfo(ctx, req.Params)
		if err != nil {
			response.Error = h.makeError(InternalError, err.Error())
		} else {
			response.Result = result
		}

	default:
		response.Error = h.makeError(MethodNotFound, fmt.Sprintf("method '%s' not found", req.Method))
	}

	return response
}

// handleTilesetFetch returns current tileset configuration
func (h *RPCHandler) handleTilesetFetch(ctx context.Context, params json.RawMessage) (interface{}, error) {
	tileset := h.webui.GetTileset()
	if tileset == nil {
		return map[string]interface{}{
			"tileset":         nil,
			"image_available": false,
		}, nil
	}

	return map[string]interface{}{
		"tileset":         tileset.ToJSON(),
		"image_available": tileset.GetImageData() != nil,
	}, nil
}

// handleGameGetState returns current game state
func (h *RPCHandler) handleGameGetState(ctx context.Context, params json.RawMessage) (interface{}, error) {
	if h.webui.view == nil {
		return map[string]interface{}{
			"state":     nil,
			"connected": false,
		}, nil
	}

	state := h.webui.view.GetCurrentState()
	return map[string]interface{}{
		"state":     state,
		"connected": true,
	}, nil
}

// GamePollParams represents parameters for game.poll method
type GamePollParams struct {
	Version uint64 `json:"version"`
	Timeout int    `json:"timeout,omitempty"`
}

// handleGamePoll implements long-polling for state changes
func (h *RPCHandler) handleGamePoll(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var pollParams GamePollParams
	if err := json.Unmarshal(params, &pollParams); err != nil {
		return nil, fmt.Errorf("invalid poll parameters: %w", err)
	}

	if h.webui.view == nil {
		return map[string]interface{}{
			"changes": nil,
			"version": 0,
		}, nil
	}

	timeout := time.Duration(pollParams.Timeout) * time.Millisecond
	if timeout <= 0 || timeout > 30*time.Second {
		timeout = 30 * time.Second
	}

	// Create context with timeout
	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stateManager := h.webui.view.stateManager
	// FIX: Pass the timeout context instead of raw timeout duration
	diff, err := stateManager.PollChangesWithContext(pollCtx, pollParams.Version)
	if err != nil {
		return nil, err
	}

	if diff == nil {
		// Timeout - no changes
		return map[string]interface{}{
			"changes": nil,
			"version": stateManager.GetCurrentVersion(),
			"timeout": true,
		}, nil
	}

	return map[string]interface{}{
		"changes": diff,
		"version": diff.Version,
		"timeout": false,
	}, nil
}

// GameInputParams represents parameters for game.sendInput method
type GameInputParams struct {
	Events []InputEvent `json:"events"`
}

// InputEvent represents a user input event
type InputEvent struct {
	Type      string `json:"type"`
	Key       string `json:"key,omitempty"`
	KeyCode   int    `json:"keyCode,omitempty"`
	Data      string `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// handleGameSendInput processes input from the client
func (h *RPCHandler) handleGameSendInput(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var inputParams GameInputParams
	if err := json.Unmarshal(params, &inputParams); err != nil {
		return nil, fmt.Errorf("invalid input parameters: %w", err)
	}

	if h.webui.view == nil {
		return nil, fmt.Errorf("no view available")
	}

	// Process each input event
	for _, event := range inputParams.Events {
		data := h.convertInputEvent(event)
		if len(data) > 0 {
			h.webui.view.SendInput(data)
		}
	}

	return map[string]interface{}{
		"processed": len(inputParams.Events),
	}, nil
}

// convertInputEvent converts web input event to terminal input
func (h *RPCHandler) convertInputEvent(event InputEvent) []byte {
	switch event.Type {
	case "keydown":
		return h.convertKeyEvent(event)
	case "paste":
		return []byte(event.Data)
	default:
		return nil
	}
}

// convertKeyEvent converts keyboard events to terminal sequences
func (h *RPCHandler) convertKeyEvent(event InputEvent) []byte {
	switch event.Key {
	case "Enter":
		return []byte("\r")
	case "Backspace":
		return []byte("\b")
	case "Tab":
		return []byte("\t")
	case "Escape":
		return []byte("\x1b")
	case "ArrowUp":
		return []byte("\x1b[A")
	case "ArrowDown":
		return []byte("\x1b[B")
	case "ArrowRight":
		return []byte("\x1b[C")
	case "ArrowLeft":
		return []byte("\x1b[D")
	case "Home":
		return []byte("\x1b[H")
	case "End":
		return []byte("\x1b[F")
	case "PageUp":
		return []byte("\x1b[5~")
	case "PageDown":
		return []byte("\x1b[6~")
	case "Delete":
		return []byte("\x1b[3~")
	case "Insert":
		return []byte("\x1b[2~")
	default:
		// Regular character
		if len(event.Key) == 1 {
			return []byte(event.Key)
		}
		return nil
	}
}

// handleTilesetUpdate updates the tileset configuration
func (h *RPCHandler) handleTilesetUpdate(ctx context.Context, params json.RawMessage) (interface{}, error) {
	// For now, return not implemented
	return nil, fmt.Errorf("tileset updates not yet implemented")
}

// handleSessionInfo returns session information
func (h *RPCHandler) handleSessionInfo(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return map[string]interface{}{
		"connected":      h.webui.view != nil,
		"timestamp":      time.Now().Unix(),
		"server_version": "1.0.0",
	}, nil
}

// makeError creates an RPC error
func (h *RPCHandler) makeError(code int, message string) *RPCError {
	return &RPCError{
		Code:    code,
		Message: message,
	}
}
