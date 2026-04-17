package trp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tx3-lang/go-sdk/sdk/core"
	"github.com/tx3-lang/go-sdk/sdk/trp"
)

func TestResolveRequestShape(t *testing.T) {
	var receivedMethod string
	var receivedParams json.RawMessage

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&req)
		receivedMethod = string(req["method"])
		receivedParams = req["params"]

		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      "1",
			"result": map[string]interface{}{
				"hash": "abc123",
				"tx":   "deadbeef",
			},
		})
	}))
	defer server.Close()

	client := trp.NewClient(trp.ClientOptions{Endpoint: server.URL})
	_, err := client.Resolve(context.Background(), trp.ResolveParams{
		Tir: core.TirEnvelope{
			Content:  "aabbcc",
			Encoding: "hex",
			Version:  "v1beta0",
		},
		Args: map[string]interface{}{"quantity": 100},
	})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify method name
	if receivedMethod != `"trp.resolve"` {
		t.Errorf("expected method 'trp.resolve', got %s", receivedMethod)
	}

	// Verify params structure
	var params map[string]interface{}
	json.Unmarshal(receivedParams, &params)
	if _, ok := params["tir"]; !ok {
		t.Error("expected 'tir' in params")
	}
	if _, ok := params["args"]; !ok {
		t.Error("expected 'args' in params")
	}
}

func TestSubmitRequestShape(t *testing.T) {
	var receivedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&req)
		receivedMethod = string(req["method"])

		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      "1",
			"result":  map[string]interface{}{"hash": "abc123"},
		})
	}))
	defer server.Close()

	client := trp.NewClient(trp.ClientOptions{Endpoint: server.URL})
	_, err := client.Submit(context.Background(), trp.SubmitParams{
		Tx: core.NewHexEnvelope("deadbeef"),
		Witnesses: []trp.TxWitness{
			{
				Key:         core.NewHexEnvelope("pubkey"),
				Signature:   core.NewHexEnvelope("sig"),
				WitnessType: "vkey",
			},
		},
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if receivedMethod != `"trp.submit"` {
		t.Errorf("expected method 'trp.submit', got %s", receivedMethod)
	}
}

func TestCheckStatusRequestShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      "1",
			"result": map[string]interface{}{
				"statuses": map[string]interface{}{
					"abc123": map[string]interface{}{
						"stage":            "confirmed",
						"confirmations":    5,
						"nonConfirmations": 0,
					},
				},
			},
		})
	}))
	defer server.Close()

	client := trp.NewClient(trp.ClientOptions{Endpoint: server.URL})
	resp, err := client.CheckStatus(context.Background(), []string{"abc123"})
	if err != nil {
		t.Fatalf("CheckStatus failed: %v", err)
	}
	status, ok := resp.Statuses["abc123"]
	if !ok {
		t.Fatal("expected status for hash 'abc123'")
	}
	if status.Stage != trp.StageConfirmed {
		t.Errorf("expected stage 'confirmed', got %s", status.Stage)
	}
}

func TestHttpErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := trp.NewClient(trp.ClientOptions{Endpoint: server.URL})
	_, err := client.Resolve(context.Background(), trp.ResolveParams{})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	var httpErr *trp.HttpError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HttpError, got %T: %v", err, err)
	}
	if httpErr.Status != 500 {
		t.Errorf("expected status 500, got %d", httpErr.Status)
	}
}

func TestJsonRpcErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      "1",
			"error": map[string]interface{}{
				"code":    -32600,
				"message": "Invalid Request",
			},
		})
	}))
	defer server.Close()

	client := trp.NewClient(trp.ClientOptions{Endpoint: server.URL})
	_, err := client.Resolve(context.Background(), trp.ResolveParams{})
	if err == nil {
		t.Fatal("expected error for JSON-RPC error")
	}
	var rpcErr *trp.GenericRpcError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected GenericRpcError, got %T: %v", err, err)
	}
	if rpcErr.Code != -32600 {
		t.Errorf("expected code -32600, got %d", rpcErr.Code)
	}
}

func TestNetworkError(t *testing.T) {
	client := trp.NewClient(trp.ClientOptions{Endpoint: "http://localhost:1"})
	_, err := client.Resolve(context.Background(), trp.ResolveParams{})
	if err == nil {
		t.Fatal("expected network error")
	}
	var netErr *trp.NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("expected NetworkError, got %T: %v", err, err)
	}
}

func TestCustomHeadersInjected(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      "1",
			"result":  map[string]interface{}{"hash": "abc", "tx": "beef"},
		})
	}))
	defer server.Close()

	client := trp.NewClient(trp.ClientOptions{
		Endpoint: server.URL,
		Headers:  map[string]string{"Authorization": "Bearer token123"},
	})
	client.Resolve(context.Background(), trp.ResolveParams{})
	if receivedAuth != "Bearer token123" {
		t.Errorf("expected Authorization header 'Bearer token123', got %q", receivedAuth)
	}
}
