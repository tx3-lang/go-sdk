package facade_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tx3-lang/go-sdk/sdk/core"
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

func newBuilder(t *testing.T, server *httptest.Server) *facade.Tx3ClientBuilder {
	t.Helper()
	return facade.FromProtocol(newTestProtocol(t)).
		TRPEndpoint(server.URL).
		WithProfile("preprod")
}

func TestBuilder_MissingEndpoint(t *testing.T) {
	_, err := facade.FromProtocol(newTestProtocol(t)).Build()
	if err == nil {
		t.Fatal("expected MissingTrpEndpointError")
	}
	var missing *facade.MissingTrpEndpointError
	if !errors.As(err, &missing) {
		t.Fatalf("expected *MissingTrpEndpointError, got %T: %v", err, err)
	}
}

func TestBuilder_UnknownProfile(t *testing.T) {
	server, _ := newMockTRPServer(t)
	defer server.Close()

	_, err := facade.FromProtocol(newTestProtocol(t)).
		TRPEndpoint(server.URL).
		WithProfile("not-a-profile").
		Build()
	if err == nil {
		t.Fatal("expected UnknownProfileError")
	}
	var profErr *tii.UnknownProfileError
	if !errors.As(err, &profErr) {
		t.Fatalf("expected *tii.UnknownProfileError, got %T: %v", err, err)
	}
}

func TestBuilder_UnknownParty(t *testing.T) {
	server, _ := newMockTRPServer(t)
	defer server.Close()

	_, err := newBuilder(t, server).
		WithParty("stranger", facade.AddressParty("addr_stranger")).
		Build()
	if err == nil {
		t.Fatal("expected UnknownPartyError")
	}
	var partyErr *facade.UnknownPartyError
	if !errors.As(err, &partyErr) {
		t.Fatalf("expected *facade.UnknownPartyError, got %T: %v", err, err)
	}
}

func TestBuilder_WithPartyUncheckedBypassesValidation(t *testing.T) {
	server, _ := newMockTRPServer(t)
	defer server.Close()

	_, err := newBuilder(t, server).
		WithPartyUnchecked("stranger", facade.AddressParty("addr_stranger")).
		Build()
	if err != nil {
		t.Fatalf("WithPartyUnchecked should not validate, got error: %v", err)
	}
}

func TestTx_UnknownTransaction(t *testing.T) {
	server, _ := newMockTRPServer(t)
	defer server.Close()

	client, err := newBuilder(t, server).
		WithParty("sender", facade.AddressParty("addr_sender")).
		WithParty("receiver", facade.AddressParty("addr_receiver")).
		WithParty("middleman", facade.AddressParty("addr_middleman")).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	_, err = client.Tx("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown transaction")
	}
	var txErr *tii.UnknownTxError
	if !errors.As(err, &txErr) {
		t.Fatalf("expected *tii.UnknownTxError, got %T: %v", err, err)
	}
}

func TestBuiltClient_WithParty_RejectsUnknown(t *testing.T) {
	server, _ := newMockTRPServer(t)
	defer server.Close()

	client, _ := newBuilder(t, server).Build()
	_, err := client.WithParty("ghost", facade.AddressParty("addr_ghost"))
	if err == nil {
		t.Fatal("expected UnknownPartyError on late-binding")
	}
	var partyErr *facade.UnknownPartyError
	if !errors.As(err, &partyErr) {
		t.Fatalf("expected *facade.UnknownPartyError, got %T: %v", err, err)
	}
}

func TestBuilder_WithEnvValue_OverridesProfileEnv(t *testing.T) {
	var receivedArgs map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&req)
		var params map[string]json.RawMessage
		json.Unmarshal(req["params"], &params)
		json.Unmarshal(params["args"], &receivedArgs)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0", "id": "1",
			"result": map[string]interface{}{"hash": "abc", "tx": "def"},
		})
	}))
	defer server.Close()

	client, err := newBuilder(t, server).
		WithPartyUnchecked("sender", facade.AddressParty("addr_sender")).
		WithPartyUnchecked("receiver", facade.AddressParty("addr_receiver")).
		WithPartyUnchecked("middleman", facade.AddressParty("addr_middleman")).
		WithEnvValue("tax", float64(999)).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	b, err := client.Tx("transfer")
	if err != nil {
		t.Fatalf("Tx failed: %v", err)
	}
	_, err = b.Arg("quantity", 100).Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if got, ok := receivedArgs["tax"].(float64); !ok || got != 999 {
		t.Errorf("expected env override tax=999, got %v", receivedArgs["tax"])
	}
}

func TestPartyAddressInjection(t *testing.T) {
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

	client, err := newBuilder(t, server).
		WithParty("sender", facade.AddressParty("addr_sender_123")).
		WithParty("receiver", facade.AddressParty("addr_receiver_456")).
		WithParty("middleman", facade.AddressParty("addr_middleman")).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	b, err := client.Tx("transfer")
	if err != nil {
		t.Fatalf("Tx failed: %v", err)
	}
	_, err = b.Arg("quantity", 100).Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if receivedArgs["sender"] != "addr_sender_123" {
		t.Errorf("expected sender address 'addr_sender_123', got %v", receivedArgs["sender"])
	}
	if receivedArgs["receiver"] != "addr_receiver_456" {
		t.Errorf("expected receiver address 'addr_receiver_456', got %v", receivedArgs["receiver"])
	}
}

func TestFullBuilderChainWithMockSigner(t *testing.T) {
	server, _ := newMockTRPServer(t)
	defer server.Close()

	mock := &mockSigner{
		address: "addr_sender",
		pubKey:  "aabbccdd",
		sig:     "11223344",
	}

	client, err := newBuilder(t, server).
		WithParty("sender", facade.SignerParty(mock)).
		WithParty("receiver", facade.AddressParty("addr_receiver")).
		WithParty("middleman", facade.AddressParty("addr_middleman")).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	ctx := context.Background()

	b, err := client.Tx("transfer")
	if err != nil {
		t.Fatalf("Tx failed: %v", err)
	}
	resolved, err := b.Arg("quantity", 10_000_000).Resolve(ctx)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if resolved.Hash == "" {
		t.Error("expected non-empty hash")
	}
	if resolved.TxHex == "" {
		t.Error("expected non-empty TxHex")
	}

	signed, err := resolved.Sign()
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if len(signed.Witnesses()) != 1 {
		t.Errorf("expected 1 witness, got %d", len(signed.Witnesses()))
	}

	submitted, err := signed.Submit(ctx)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if submitted.Hash == "" {
		t.Error("expected non-empty submitted hash")
	}

	status, err := submitted.WaitForConfirmed(ctx, facade.DefaultPollConfig())
	if err != nil {
		t.Fatalf("WaitForConfirmed failed: %v", err)
	}
	if status.Stage != trp.StageConfirmed {
		t.Errorf("expected stage 'confirmed', got %s", status.Stage)
	}
}

func TestFromParts_CodegenFlow(t *testing.T) {
	var receivedArgs map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&req)
		var params map[string]json.RawMessage
		json.Unmarshal(req["params"], &params)
		json.Unmarshal(params["args"], &receivedArgs)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0", "id": "1",
			"result": map[string]interface{}{"hash": "abc", "tx": "def"},
		})
	}))
	defer server.Close()

	protocol := newTestProtocol(t)
	transactions := map[string]core.TirEnvelope{}
	for name, tx := range protocol.Transactions() {
		transactions[name] = tx.Tir
	}

	client, err := facade.FromParts(transactions, nil, nil).
		TRPEndpoint(server.URL).
		WithPartyUnchecked("sender", facade.AddressParty("addr_sender_codegen")).
		WithPartyUnchecked("receiver", facade.AddressParty("addr_receiver_codegen")).
		WithPartyUnchecked("middleman", facade.AddressParty("addr_middleman")).
		Build()
	if err != nil {
		t.Fatalf("FromParts build failed: %v", err)
	}

	b, err := client.Tx("transfer")
	if err != nil {
		t.Fatalf("Tx failed: %v", err)
	}
	if _, err := b.Arg("quantity", 1).Resolve(context.Background()); err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if receivedArgs["sender"] != "addr_sender_codegen" {
		t.Errorf("expected sender injected via FromParts/WithPartyUnchecked, got %v", receivedArgs["sender"])
	}
}

func TestSubmitHashMismatch(t *testing.T) {
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

	mock := &mockSigner{address: "addr", pubKey: "aa", sig: "bb"}
	client, err := facade.FromProtocol(newTestProtocol(t)).
		TRPEndpoint(server.URL).
		WithParty("sender", facade.SignerParty(mock)).
		WithParty("receiver", facade.AddressParty("addr_r")).
		WithParty("middleman", facade.AddressParty("addr_m")).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	b, _ := client.Tx("transfer")
	resolved, _ := b.Arg("quantity", 100).Resolve(context.Background())
	signed, _ := resolved.Sign()
	_, err = signed.Submit(context.Background())

	if err == nil {
		t.Fatal("expected SubmitHashMismatchError")
	}
	var mismatchErr *facade.SubmitHashMismatchError
	if !errors.As(err, &mismatchErr) {
		t.Fatalf("expected SubmitHashMismatchError, got %T: %v", err, err)
	}
}

func TestWaitForConfirmedTimeout(t *testing.T) {
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

	mock := &mockSigner{address: "addr", pubKey: "aa", sig: "bb"}
	client, _ := facade.FromProtocol(newTestProtocol(t)).
		TRPEndpoint(server.URL).
		WithParty("sender", facade.SignerParty(mock)).
		WithParty("receiver", facade.AddressParty("addr_r")).
		WithParty("middleman", facade.AddressParty("addr_m")).
		Build()

	ctx := context.Background()
	b, _ := client.Tx("transfer")
	resolved, _ := b.Arg("quantity", 100).Resolve(ctx)
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

	mock := &mockSigner{address: "addr", pubKey: "aa", sig: "bb"}
	client, _ := facade.FromProtocol(newTestProtocol(t)).
		TRPEndpoint(server.URL).
		WithParty("sender", facade.SignerParty(mock)).
		WithParty("receiver", facade.AddressParty("addr_r")).
		WithParty("middleman", facade.AddressParty("addr_m")).
		Build()

	ctx := context.Background()
	b, _ := client.Tx("transfer")
	resolved, _ := b.Arg("quantity", 100).Resolve(ctx)
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

func (m *mockSigner) Sign(_ signer.SignRequest) (*signer.TxWitness, error) {
	return signer.NewVKeyWitness(m.pubKey, m.sig), nil
}

func init() {
	var _ signer.Signer = &mockSigner{}
}
