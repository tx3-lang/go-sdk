// Package trp provides a Transaction Resolution Protocol (TRP) client for Go
package trp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// TirInfo contains the Transaction Intermediate Representation information
type TirInfo struct {
	Version  string `json:"version"`
	Bytecode string `json:"bytecode"`
	Encoding string `json:"encoding"` // "base64" | "hex" | string
}

// TxEnvelope represents a resolved transaction envelope
type TxEnvelope struct {
	Tx   string `json:"tx"`
	Hash string `json:"hash"`
}

// ClientOptions configures the TRP client
type ClientOptions struct {
	Endpoint string
	Headers  map[string]string
	EnvArgs  map[string]interface{}
	Timeout  time.Duration
}

// ProtoTxRequest represents a prototype transaction to be resolved
type ProtoTxRequest struct {
	Tir  TirInfo     `json:"tir"`
	Args interface{} `json:"args"`
}

// jsonRPCRequest represents a JSON-RPC request
type jsonRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  jsonRPCParams `json:"params"`
	ID      string        `json:"id"`
}

// jsonRPCParams represents the parameters for a TRP resolution request
type jsonRPCParams struct {
	Tir  TirInfo     `json:"tir"`
	Args interface{} `json:"args"`
	Env  interface{} `json:"env,omitempty"`
}

// jsonRPCResponse represents a JSON-RPC response
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
	ID      string          `json:"id"`
}

// jsonRPCError represents a JSON-RPC error
type jsonRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error is a custom error type for TRP operations
type Error struct {
	Message string
	Data    interface{}
}

// Error implements the error interface for TRPError
func (e *Error) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Data)
	}
	return e.Message
}

// Client is a client for interacting with a TRP server
type Client struct {
	options    ClientOptions
	httpClient *http.Client
}

// NewClient creates a new TRPClient with the given options
func NewClient(options ClientOptions) *Client {
	// Set default timeout if not provided
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		options: options,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Resolve sends a transaction to be resolved by the TRP server
func (c *Client) Resolve(protoTx ProtoTxRequest) (*TxEnvelope, error) {
	// Create JSON-RPC request
	params := jsonRPCParams{
		Tir:  protoTx.Tir,
		Args: protoTx.Args,
	}

	// Add environment arguments if provided
	if len(c.options.EnvArgs) > 0 {
		params.Env = c.options.EnvArgs
	}

	requestID := uuid.New().String()
	rpcRequest := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "trp.resolve",
		Params:  params,
		ID:      requestID,
	}

	// Marshal request to JSON
	reqBody, err := json.Marshal(rpcRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", c.options.Endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set content type header
	req.Header.Set("Content-Type", "application/json")

	// Add custom headers if provided
	for key, value := range c.options.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	// Read and parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResponse jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for JSON-RPC error
	if rpcResponse.Error != nil {
		return nil, &Error{
			Message: rpcResponse.Error.Message,
			Data:    rpcResponse.Error.Data,
		}
	}

	// Parse result
	var result TxEnvelope
	if err := json.Unmarshal(rpcResponse.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}
