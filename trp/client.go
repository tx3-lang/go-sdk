package trp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ClientOptions configures a TRP client.
type ClientOptions struct {
	Endpoint string            // Full URL of the TRP JSON-RPC server
	Headers  map[string]string // Optional custom headers for every request
	Timeout  time.Duration     // HTTP request timeout (default: 30s)
}

// Client is a low-level TRP JSON-RPC client.
type Client struct {
	options    ClientOptions
	httpClient *http.Client
}

// NewClient creates a new TRP client with the given options.
func NewClient(options ClientOptions) *Client {
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

// jsonRPCRequest is a JSON-RPC 2.0 request envelope.
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response envelope.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// call executes a JSON-RPC method and returns the raw result.
func (c *Client) call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      uuid.New().String(),
		Method:  method,
		Params:  params,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &NetworkError{Cause: fmt.Errorf("failed to marshal request: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.options.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.options.Headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &NetworkError{Cause: fmt.Errorf("failed to read response body: %w", err)}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &HttpError{
			Status:     resp.StatusCode,
			StatusText: resp.Status,
			Body:       string(respBody),
		}
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, &DeserializationError{Cause: err, Raw: string(respBody)}
	}

	if rpcResp.Error != nil {
		return nil, classifyRpcError(rpcResp.Error)
	}

	if rpcResp.Result == nil {
		return nil, &MalformedResponseError{Detail: "response has no result"}
	}

	return rpcResp.Result, nil
}

// classifyRpcError maps a JSON-RPC error to a typed TRP error.
func classifyRpcError(e *rpcError) error {
	// Try to parse structured diagnostic data
	if e.Data != nil {
		var diag map[string]interface{}
		if json.Unmarshal(e.Data, &diag) == nil {
			if kind, ok := diag["kind"].(string); ok {
				switch kind {
				case "UnsupportedTir":
					return &UnsupportedTirError{
						Expected: stringFromMap(diag, "expected"),
						Provided: stringFromMap(diag, "provided"),
					}
				case "MissingTxArg":
					return &MissingTxArgError{
						Key:     stringFromMap(diag, "key"),
						ArgType: stringFromMap(diag, "argType"),
					}
				case "InputNotResolved":
					return &InputNotResolvedError{
						Name: stringFromMap(diag, "name"),
					}
				case "TxScriptFailure":
					var logs []string
					if logsRaw, ok := diag["logs"]; ok {
						if logsArr, ok := logsRaw.([]interface{}); ok {
							for _, l := range logsArr {
								if s, ok := l.(string); ok {
									logs = append(logs, s)
								}
							}
						}
					}
					return &TxScriptFailureError{Logs: logs}
				case "InvalidTirEnvelope":
					return &InvalidTirEnvelopeError{}
				case "InvalidTirBytes":
					return &InvalidTirBytesError{}
				case "UnsupportedEra":
					return &UnsupportedEraError{Era: stringFromMap(diag, "era")}
				}
			}
		}
	}
	return &GenericRpcError{
		Code:    e.Code,
		Message: e.Message,
		Data:    e.Data,
	}
}

func stringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Resolve invokes the trp.resolve JSON-RPC method.
func (c *Client) Resolve(ctx context.Context, params ResolveParams) (*TxEnvelope, error) {
	result, err := c.call(ctx, "trp.resolve", params)
	if err != nil {
		return nil, err
	}
	var envelope TxEnvelope
	if err := json.Unmarshal(result, &envelope); err != nil {
		return nil, &DeserializationError{Cause: err, Raw: string(result)}
	}
	return &envelope, nil
}

// Submit invokes the trp.submit JSON-RPC method.
func (c *Client) Submit(ctx context.Context, params SubmitParams) (*SubmitResponse, error) {
	result, err := c.call(ctx, "trp.submit", params)
	if err != nil {
		return nil, err
	}
	var resp SubmitResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, &DeserializationError{Cause: err, Raw: string(result)}
	}
	return &resp, nil
}

// CheckStatus invokes the trp.checkStatus JSON-RPC method.
func (c *Client) CheckStatus(ctx context.Context, hashes []string) (*CheckStatusResponse, error) {
	params := CheckStatusParams{Hashes: hashes}
	result, err := c.call(ctx, "trp.checkStatus", params)
	if err != nil {
		return nil, err
	}
	var resp CheckStatusResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, &DeserializationError{Cause: err, Raw: string(result)}
	}
	return &resp, nil
}
