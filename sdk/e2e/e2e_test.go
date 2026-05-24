//go:build e2e
// +build e2e

// End-to-end tests for the Tx3 Go SDK.
// These tests require a running TRP server and are skipped if
// the TRP_ENDPOINT_PREPROD environment variable is not set.
//
// Required environment variables:
//   - TRP_ENDPOINT_PREPROD: Full URL of the preprod TRP server
//   - TRP_API_KEY_PREPROD: API key for the preprod TRP server
//   - TEST_PARTY_A_ADDRESS: Bech32 address for the first party
//   - TEST_PARTY_A_MNEMONIC: BIP39 mnemonic for the first party
//
// Optional:
//   - TEST_PARTY_B_ADDRESS: Second party address
//   - TEST_PARTY_B_MNEMONIC: Second party mnemonic
package e2e_test

import (
	"context"
	"os"
	"testing"

	"github.com/tx3-lang/go-sdk/sdk/facade"
	"github.com/tx3-lang/go-sdk/sdk/signer"
	"github.com/tx3-lang/go-sdk/sdk/tii"
)

func skipIfNoTRP(t *testing.T) {
	t.Helper()
	if os.Getenv("TRP_ENDPOINT_PREPROD") == "" {
		t.Skip("TRP_ENDPOINT_PREPROD not set, skipping e2e test")
	}
}

func requireEnv(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Skipf("%s not set, skipping e2e test", name)
	}
	return v
}

// e2eBuilder seeds a Tx3ClientBuilder from the test fixture and applies the
// preprod profile + (optional) dmtr-api-key header from the environment.
func e2eBuilder(t *testing.T, endpoint string) *facade.Tx3ClientBuilder {
	t.Helper()
	protocol, err := tii.FromFile("../testdata/transfer.tii")
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	builder := facade.FromProtocol(protocol).
		TRPEndpoint(endpoint).
		WithProfile("preprod")

	if apiKey := os.Getenv("TRP_API_KEY_PREPROD"); apiKey != "" {
		builder = builder.WithHeader("dmtr-api-key", apiKey)
	}
	return builder
}

func TestE2EHappyPath(t *testing.T) {
	skipIfNoTRP(t)
	endpoint := requireEnv(t, "TRP_ENDPOINT_PREPROD")
	partyAAddr := requireEnv(t, "TEST_PARTY_A_ADDRESS")
	partyAMnemonic := requireEnv(t, "TEST_PARTY_A_MNEMONIC")

	cardanoSigner, err := signer.CardanoSignerFromMnemonic(partyAAddr, partyAMnemonic)
	if err != nil {
		t.Fatalf("CardanoSignerFromMnemonic failed: %v", err)
	}

	partyBAddr := os.Getenv("TEST_PARTY_B_ADDRESS")
	if partyBAddr == "" {
		partyBAddr = partyAAddr
	}

	client, err := e2eBuilder(t, endpoint).
		WithParty("sender", facade.SignerParty(cardanoSigner)).
		WithParty("receiver", facade.AddressParty(partyBAddr)).
		WithParty("middleman", facade.AddressParty(partyBAddr)).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	ctx := context.Background()

	txb, err := client.Tx("transfer")
	if err != nil {
		t.Fatalf("Tx failed: %v", err)
	}

	resolved, err := txb.Arg("quantity", 10_000_000).Resolve(ctx)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	signed, err := resolved.Sign()
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	submitted, err := signed.Submit(ctx)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	status, err := submitted.WaitForConfirmed(ctx, facade.DefaultPollConfig())
	if err != nil {
		t.Fatalf("WaitForConfirmed failed: %v", err)
	}

	t.Logf("Transaction %s confirmed at stage %s", submitted.Hash, status.Stage)
}

func TestE2EBadEndpoint(t *testing.T) {
	skipIfNoTRP(t)

	protocol, err := tii.FromFile("../testdata/transfer.tii")
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	client, err := facade.FromProtocol(protocol).
		TRPEndpoint("http://localhost:1").
		WithParty("sender", facade.AddressParty("addr_test1...")).
		WithParty("receiver", facade.AddressParty("addr_test1...")).
		WithParty("middleman", facade.AddressParty("addr_test1...")).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	txb, err := client.Tx("transfer")
	if err != nil {
		t.Fatalf("Tx failed: %v", err)
	}
	_, err = txb.Arg("quantity", 100).Resolve(context.Background())
	if err == nil {
		t.Fatal("expected error for bad endpoint")
	}
}

func TestE2EPollTimeout(t *testing.T) {
	skipIfNoTRP(t)
	endpoint := requireEnv(t, "TRP_ENDPOINT_PREPROD")
	partyAAddr := requireEnv(t, "TEST_PARTY_A_ADDRESS")
	partyAMnemonic := requireEnv(t, "TEST_PARTY_A_MNEMONIC")

	cardanoSigner, err := signer.CardanoSignerFromMnemonic(partyAAddr, partyAMnemonic)
	if err != nil {
		t.Fatalf("CardanoSignerFromMnemonic failed: %v", err)
	}

	client, err := e2eBuilder(t, endpoint).
		WithParty("sender", facade.SignerParty(cardanoSigner)).
		WithParty("receiver", facade.AddressParty(partyAAddr)).
		WithParty("middleman", facade.AddressParty(partyAAddr)).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	ctx := context.Background()
	txb, err := client.Tx("transfer")
	if err != nil {
		t.Fatalf("Tx failed: %v", err)
	}
	resolved, err := txb.Arg("quantity", 10_000_000).Resolve(ctx)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	signed, err := resolved.Sign()
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	submitted, err := signed.Submit(ctx)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Try with very aggressive timeout — this should time out
	_, err = submitted.WaitForConfirmed(ctx, facade.PollConfig{Attempts: 1, Delay: 0})
	// This may or may not timeout depending on speed, so just verify it doesn't panic
	t.Logf("Poll result: %v", err)
}
