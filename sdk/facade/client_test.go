package facade_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tx3-lang/go-sdk/sdk/facade"
	"github.com/tx3-lang/go-sdk/sdk/signer"
	"github.com/tx3-lang/go-sdk/sdk/tii"
	"github.com/tx3-lang/go-sdk/sdk/trp"
)

func newTestProtocol(t *testing.T) *tii.Protocol {
	t.Helper()
	p, err := tii.FromFile("../testdata/transfer.tii")
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}
	return p
}

func newMockTRPServer(t *testing.T) (*httptest.Server, *trp.Client) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&req)
		method := ""
		json.Unmarshal(req["method"], &method)

		switch method {
		case "trp.resolve":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "1",
				"result": map[string]interface{}{
					"hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"tx":   "deadbeefcafebabe",
				},
			})
		case "trp.submit":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "1",
				"result": map[string]interface{}{
					"hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				},
			})
		case "trp.checkStatus":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "1",
				"result": map[string]interface{}{
					"statuses": map[string]interface{}{
						"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855": map[string]interface{}{
							"stage":            "confirmed",
							"confirmations":    3,
							"nonConfirmations": 0,
						},
					},
				},
			})
		}
	}))
	client := trp.NewClient(trp.ClientOptions{Endpoint: server.URL})
	return server, client
}

func TestBuilderUnknownTransaction(t *testing.T) {
	protocol := newTestProtocol(t)
	server, trpClient := newMockTRPServer(t)
	defer server.Close()

	client := facade.NewClient(protocol, trpClient)
	_, err := client.Tx("nonexistent").
		Arg("quantity", 100).
		Resolve(context.Background())

	if err == nil {
		t.Fatal("expected error for unknown transaction")
	}
	var txErr *tii.UnknownTxError
	if !errors.As(err, &txErr) {
		t.Fatalf("expected UnknownTxError, got %T: %v", err, err)
	}
}

func TestBuilderUnknownParty(t *testing.T) {
	protocol := newTestProtocol(t)
	server, trpClient := newMockTRPServer(t)
	defer server.Close()

	client := facade.NewClient(protocol, trpClient).
		WithParty("unknown_party", facade.AddressParty("addr_test1..."))

	_, err := client.Tx("transfer").
		Arg("quantity", 100).
		Resolve(context.Background())

	if err == nil {
		t.Fatal("expected error for unknown party")
	}
	var partyErr *facade.UnknownPartyError
	if !errors.As(err, &partyErr) {
		t.Fatalf("expected UnknownPartyError, got %T: %v", err, err)
	}
	if partyErr.Name != "unknown_party" {
		t.Errorf("expected party name 'unknown_party', got %q", partyErr.Name)
	}
}

func TestPartyAddressInjection(t *testing.T) {
	protocol := newTestProtocol(t)

	var receivedArgs map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&req)
		var params map[string]json.RawMessage
		json.Unmarshal(req["params"], &params)
		json.Unmarshal(params["args"], &receivedArgs)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      "1",
			"result":  map[string]interface{}{"hash": "abc", "tx": "def"},
		})
	}))
	defer server.Close()

	trpClient := trp.NewClient(trp.ClientOptions{Endpoint: server.URL})
	client := facade.NewClient(protocol, trpClient).
		WithParty("sender", facade.AddressParty("addr_sender_123")).
		WithParty("receiver", facade.AddressParty("addr_receiver_456"))

	_, err := client.Tx("transfer").
		Arg("quantity", 100).
		Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify party addresses were injected into args
	if receivedArgs["sender"] != "addr_sender_123" {
		t.Errorf("expected sender address 'addr_sender_123', got %v", receivedArgs["sender"])
	}
	if receivedArgs["receiver"] != "addr_receiver_456" {
		t.Errorf("expected receiver address 'addr_receiver_456', got %v", receivedArgs["receiver"])
	}
}

func TestFullBuilderChainWithMockSigner(t *testing.T) {
	protocol := newTestProtocol(t)
	server, trpClient := newMockTRPServer(t)
	defer server.Close()

	mock := &mockSigner{
		address: "addr_sender",
		pubKey:  "aabbccdd",
		sig:     "11223344",
	}

	client := facade.NewClient(protocol, trpClient).
		WithProfile("preprod").
		WithParty("sender", facade.SignerParty(mock)).
		WithParty("receiver", facade.AddressParty("addr_receiver")).
		WithParty("middleman", facade.AddressParty("addr_middleman"))

	ctx := context.Background()

	// Resolve
	resolved, err := client.Tx("transfer").
		Arg("quantity", 10_000_000).
		Resolve(ctx)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if resolved.Hash == "" {
		t.Error("expected non-empty hash")
	}
	if resolved.TxHex == "" {
		t.Error("expected non-empty TxHex")
	}

	// Sign
	signed, err := resolved.Sign()
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if len(signed.Witnesses()) != 1 {
		t.Errorf("expected 1 witness, got %d", len(signed.Witnesses()))
	}

	// Submit
	submitted, err := signed.Submit(ctx)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if submitted.Hash == "" {
		t.Error("expected non-empty submitted hash")
	}

	// WaitForConfirmed
	status, err := submitted.WaitForConfirmed(ctx, facade.DefaultPollConfig())
	if err != nil {
		t.Fatalf("WaitForConfirmed failed: %v", err)
	}
	if status.Stage != trp.StageConfirmed {
		t.Errorf("expected stage 'confirmed', got %s", status.Stage)
	}
}

func TestSubmitHashMismatch(t *testing.T) {
	protocol := newTestProtocol(t)

	// Server returns a different hash on submit
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&req)
		method := ""
		json.Unmarshal(req["method"], &method)

		switch method {
		case "trp.resolve":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "1",
				"result": map[string]interface{}{
					"hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"tx":   "deadbeef",
				},
			})
		case "trp.submit":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "1",
				"result":  map[string]interface{}{"hash": "different_hash"},
			})
		}
	}))
	defer server.Close()

	trpClient := trp.NewClient(trp.ClientOptions{Endpoint: server.URL})
	mock := &mockSigner{address: "addr", pubKey: "aa", sig: "bb"}
	client := facade.NewClient(protocol, trpClient).
		WithParty("sender", facade.SignerParty(mock)).
		WithParty("receiver", facade.AddressParty("addr_r")).
		WithParty("middleman", facade.AddressParty("addr_m"))

	resolved, _ := client.Tx("transfer").Arg("quantity", 100).Resolve(context.Background())
	signed, _ := resolved.Sign()
	_, err := signed.Submit(context.Background())

	if err == nil {
		t.Fatal("expected SubmitHashMismatchError")
	}
	var mismatchErr *facade.SubmitHashMismatchError
	if !errors.As(err, &mismatchErr) {
		t.Fatalf("expected SubmitHashMismatchError, got %T: %v", err, err)
	}
}

func TestWaitForConfirmedTimeout(t *testing.T) {
	protocol := newTestProtocol(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&req)
		method := ""
		json.Unmarshal(req["method"], &method)

		switch method {
		case "trp.resolve":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "1",
				"result":  map[string]interface{}{"hash": "abc", "tx": "def"},
			})
		case "trp.submit":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "1",
				"result":  map[string]interface{}{"hash": "abc"},
			})
		case "trp.checkStatus":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "1",
				"result": map[string]interface{}{
					"statuses": map[string]interface{}{
						"abc": map[string]interface{}{
							"stage":            "pending",
							"confirmations":    0,
							"nonConfirmations": 0,
						},
					},
				},
			})
		}
	}))
	defer server.Close()

	trpClient := trp.NewClient(trp.ClientOptions{Endpoint: server.URL})
	mock := &mockSigner{address: "addr", pubKey: "aa", sig: "bb"}
	client := facade.NewClient(protocol, trpClient).
		WithParty("sender", facade.SignerParty(mock)).
		WithParty("receiver", facade.AddressParty("addr_r")).
		WithParty("middleman", facade.AddressParty("addr_m"))

	ctx := context.Background()
	resolved, _ := client.Tx("transfer").Arg("quantity", 100).Resolve(ctx)
	signed, _ := resolved.Sign()
	submitted, _ := signed.Submit(ctx)

	_, err := submitted.WaitForConfirmed(ctx, facade.PollConfig{Attempts: 1, Delay: 0})
	if err == nil {
		t.Fatal("expected FinalizedTimeoutError")
	}
	var timeoutErr *facade.FinalizedTimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected FinalizedTimeoutError, got %T: %v", err, err)
	}
}

func TestWaitForConfirmedDropped(t *testing.T) {
	protocol := newTestProtocol(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&req)
		method := ""
		json.Unmarshal(req["method"], &method)

		switch method {
		case "trp.resolve":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": "1",
				"result": map[string]interface{}{"hash": "abc", "tx": "def"},
			})
		case "trp.submit":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": "1",
				"result": map[string]interface{}{"hash": "abc"},
			})
		case "trp.checkStatus":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": "1",
				"result": map[string]interface{}{
					"statuses": map[string]interface{}{
						"abc": map[string]interface{}{
							"stage":            "dropped",
							"confirmations":    0,
							"nonConfirmations": 0,
						},
					},
				},
			})
		}
	}))
	defer server.Close()

	trpClient := trp.NewClient(trp.ClientOptions{Endpoint: server.URL})
	mock := &mockSigner{address: "addr", pubKey: "aa", sig: "bb"}
	client := facade.NewClient(protocol, trpClient).
		WithParty("sender", facade.SignerParty(mock)).
		WithParty("receiver", facade.AddressParty("addr_r")).
		WithParty("middleman", facade.AddressParty("addr_m"))

	ctx := context.Background()
	resolved, _ := client.Tx("transfer").Arg("quantity", 100).Resolve(ctx)
	signed, _ := resolved.Sign()
	submitted, _ := signed.Submit(ctx)

	_, err := submitted.WaitForConfirmed(ctx, facade.DefaultPollConfig())
	if err == nil {
		t.Fatal("expected FinalizedFailedError")
	}
	var failedErr *facade.FinalizedFailedError
	if !errors.As(err, &failedErr) {
		t.Fatalf("expected FinalizedFailedError, got %T: %v", err, err)
	}
	if failedErr.Stage != "dropped" {
		t.Errorf("expected stage 'dropped', got %q", failedErr.Stage)
	}
}

type mockSigner struct {
	address string
	pubKey  string
	sig     string
}

func (m *mockSigner) Address() string { return m.address }

func (m *mockSigner) Sign(_ string) (*signer.TxWitness, error) {
	return signer.NewVKeyWitness(m.pubKey, m.sig), nil
}

func init() {
	var _ signer.Signer = &mockSigner{}
}
