package facade_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tx3-lang/go-sdk/sdk/core"
	"github.com/tx3-lang/go-sdk/sdk/facade"
	"github.com/tx3-lang/go-sdk/sdk/trp"
)

func fakeWitness(pubHex, sigHex string) trp.TxWitness {
	return trp.TxWitness{
		Key:         core.NewHexEnvelope(pubHex),
		Signature:   core.NewHexEnvelope(sigHex),
		WitnessType: "vkey",
	}
}

// recordingTRPServer captures the SubmitParams from the most recent submit call.
func recordingTRPServer(t *testing.T) (*httptest.Server, *trp.Client, *trp.SubmitParams) {
	t.Helper()
	var captured trp.SubmitParams
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&req)
		method := ""
		json.Unmarshal(req["method"], &method)

		switch method {
		case "trp.resolve":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": "1",
				"result": map[string]interface{}{
					"hash": "deadbeef00000000000000000000000000000000000000000000000000000000",
					"tx":   "84a40081",
				},
			})
		case "trp.submit":
			var params trp.SubmitParams
			json.Unmarshal(req["params"], &params)
			captured = params
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": "1",
				"result": map[string]interface{}{
					"hash": "deadbeef00000000000000000000000000000000000000000000000000000000",
				},
			})
		}
	}))
	client := trp.NewClient(trp.ClientOptions{Endpoint: server.URL})
	return server, client, &captured
}

func TestResolvedTx_AddWitness_OnlyNoSigners(t *testing.T) {
	server, _, captured := recordingTRPServer(t)
	defer server.Close()

	client, err := facade.FromProtocol(newTestProtocol(t)).
		TRPEndpoint(server.URL).
		WithParty("sender", facade.AddressParty("addr_sender")).
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
	resolved, err := b.Arg("quantity", 100).Resolve(ctx)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	signed, err := resolved.AddWitness(fakeWitness("aa", "bb")).Sign()
	if err != nil {
		t.Fatalf("Sign with manual witness only must succeed: %v", err)
	}

	if _, err := signed.Submit(ctx); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	if got := len(captured.Witnesses); got != 1 {
		t.Fatalf("expected 1 witness submitted, got %d", got)
	}
	if got := captured.Witnesses[0].Key.Content; got != "aa" {
		t.Errorf("expected key 'aa', got %q", got)
	}
}

func TestResolvedTx_AddWitness_MixedWithRegisteredSigner(t *testing.T) {
	server, _, captured := recordingTRPServer(t)
	defer server.Close()

	mock := &mockSigner{address: "addr_sender", pubKey: "11", sig: "22"}
	client, err := facade.FromProtocol(newTestProtocol(t)).
		TRPEndpoint(server.URL).
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
	resolved, err := b.Arg("quantity", 100).Resolve(ctx)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	signed, err := resolved.AddWitness(fakeWitness("aa", "bb")).Sign()
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if _, err := signed.Submit(ctx); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	if got := len(captured.Witnesses); got != 2 {
		t.Fatalf("expected 2 witnesses, got %d", got)
	}
	if captured.Witnesses[0].Key.Content != "11" {
		t.Errorf("expected registered witness key '11' first, got %q", captured.Witnesses[0].Key.Content)
	}
	if captured.Witnesses[1].Key.Content != "aa" {
		t.Errorf("expected manual witness key 'aa' second, got %q", captured.Witnesses[1].Key.Content)
	}
}

func TestResolvedTx_AddWitness_PreservesOrder(t *testing.T) {
	server, _, captured := recordingTRPServer(t)
	defer server.Close()

	client, err := facade.FromProtocol(newTestProtocol(t)).
		TRPEndpoint(server.URL).
		WithParty("sender", facade.AddressParty("addr_sender")).
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
	resolved, err := b.Arg("quantity", 100).Resolve(ctx)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	signed, err := resolved.
		AddWitness(fakeWitness("01", "10")).
		AddWitness(fakeWitness("02", "20")).
		AddWitness(fakeWitness("03", "30")).
		Sign()
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if _, err := signed.Submit(ctx); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	gotKeys := make([]string, len(captured.Witnesses))
	for i, w := range captured.Witnesses {
		gotKeys[i] = w.Key.Content
	}
	wantKeys := []string{"01", "02", "03"}
	for i, want := range wantKeys {
		if i >= len(gotKeys) || gotKeys[i] != want {
			t.Errorf("witness order mismatch: got %v, want %v", gotKeys, wantKeys)
			break
		}
	}
}
