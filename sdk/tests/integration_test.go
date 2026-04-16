// Integration tests for the Tx3 Go SDK.
// These tests require a running TRP server and are skipped if
// the TRP_ENDPOINT environment variable is not set.
//
// Required environment variables:
//   - TRP_ENDPOINT: Full URL of the TRP server
//   - TEST_PARTY_A_ADDRESS: Bech32 address for the first party
//   - TEST_PARTY_A_MNEMONIC: BIP39 mnemonic for the first party
//
// Optional:
//   - TEST_PARTY_B_ADDRESS: Second party address
//   - TEST_PARTY_B_MNEMONIC: Second party mnemonic
package tests

import (
	"context"
	"os"
	"testing"

	"github.com/tx3-lang/go-sdk/sdk/facade"
	"github.com/tx3-lang/go-sdk/sdk/signer"
	"github.com/tx3-lang/go-sdk/sdk/tii"
	"github.com/tx3-lang/go-sdk/sdk/trp"
)

func skipIfNoTRP(t *testing.T) {
	t.Helper()
	if os.Getenv("TRP_ENDPOINT") == "" {
		t.Skip("TRP_ENDPOINT not set, skipping integration test")
	}
}

func requireEnv(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Skipf("%s not set, skipping integration test", name)
	}
	return v
}

func TestIntegrationHappyPath(t *testing.T) {
	skipIfNoTRP(t)
	endpoint := requireEnv(t, "TRP_ENDPOINT")
	partyAAddr := requireEnv(t, "TEST_PARTY_A_ADDRESS")
	partyAMnemonic := requireEnv(t, "TEST_PARTY_A_MNEMONIC")

	protocol, err := tii.FromFile("fixtures/transfer.tii")
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	trpClient := trp.NewClient(trp.ClientOptions{Endpoint: endpoint})

	cardanoSigner, err := signer.CardanoSignerFromMnemonic(partyAAddr, partyAMnemonic)
	if err != nil {
		t.Fatalf("CardanoSignerFromMnemonic failed: %v", err)
	}

	partyBAddr := os.Getenv("TEST_PARTY_B_ADDRESS")
	if partyBAddr == "" {
		partyBAddr = partyAAddr
	}

	client := facade.NewClient(protocol, trpClient).
		WithProfile("preprod").
		WithParty("sender", facade.SignerParty(cardanoSigner)).
		WithParty("receiver", facade.AddressParty(partyBAddr)).
		WithParty("middleman", facade.AddressParty(partyBAddr))

	ctx := context.Background()

	resolved, err := client.Tx("transfer").
		Arg("quantity", 10_000_000).
		Resolve(ctx)
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

func TestIntegrationBadEndpoint(t *testing.T) {
	skipIfNoTRP(t)

	protocol, err := tii.FromFile("fixtures/transfer.tii")
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	trpClient := trp.NewClient(trp.ClientOptions{Endpoint: "http://localhost:1"})
	client := facade.NewClient(protocol, trpClient).
		WithParty("sender", facade.AddressParty("addr_test1...")).
		WithParty("receiver", facade.AddressParty("addr_test1...")).
		WithParty("middleman", facade.AddressParty("addr_test1..."))

	_, err = client.Tx("transfer").
		Arg("quantity", 100).
		Resolve(context.Background())
	if err == nil {
		t.Fatal("expected error for bad endpoint")
	}
}

func TestIntegrationPollTimeout(t *testing.T) {
	skipIfNoTRP(t)
	endpoint := requireEnv(t, "TRP_ENDPOINT")
	partyAAddr := requireEnv(t, "TEST_PARTY_A_ADDRESS")
	partyAMnemonic := requireEnv(t, "TEST_PARTY_A_MNEMONIC")

	protocol, err := tii.FromFile("fixtures/transfer.tii")
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	trpClient := trp.NewClient(trp.ClientOptions{Endpoint: endpoint})
	cardanoSigner, err := signer.CardanoSignerFromMnemonic(partyAAddr, partyAMnemonic)
	if err != nil {
		t.Fatalf("CardanoSignerFromMnemonic failed: %v", err)
	}

	client := facade.NewClient(protocol, trpClient).
		WithProfile("preprod").
		WithParty("sender", facade.SignerParty(cardanoSigner)).
		WithParty("receiver", facade.AddressParty(partyAAddr)).
		WithParty("middleman", facade.AddressParty(partyAAddr))

	ctx := context.Background()
	resolved, err := client.Tx("transfer").Arg("quantity", 10_000_000).Resolve(ctx)
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
