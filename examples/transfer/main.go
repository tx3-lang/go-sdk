// Example: complete transfer transaction using the Tx3 Go SDK.
//
// This example loads a .tii protocol file, configures a TRP client,
// attaches parties with signers, and executes a transfer transaction
// through the full lifecycle: resolve -> sign -> submit -> confirm.
//
// Usage:
//
//	export TRP_ENDPOINT=http://localhost:3000
//	export SENDER_ADDRESS=addr_test1...
//	export SENDER_MNEMONIC="word1 word2 ... word24"
//	export RECEIVER_ADDRESS=addr_test1...
//	go run .
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	tx3 "github.com/tx3-lang/go-sdk"
	"github.com/tx3-lang/go-sdk/signer"
	"github.com/tx3-lang/go-sdk/trp"
)

func main() {
	endpoint := os.Getenv("TRP_ENDPOINT")
	senderAddr := os.Getenv("SENDER_ADDRESS")
	senderMnemonic := os.Getenv("SENDER_MNEMONIC")
	receiverAddr := os.Getenv("RECEIVER_ADDRESS")

	if endpoint == "" || senderAddr == "" || senderMnemonic == "" || receiverAddr == "" {
		log.Fatal("Set TRP_ENDPOINT, SENDER_ADDRESS, SENDER_MNEMONIC, and RECEIVER_ADDRESS")
	}

	// 1. Load protocol from .tii file
	protocol, err := tx3.ProtocolFromFile("../../tests/fixtures/transfer.tii")
	if err != nil {
		log.Fatalf("Failed to load protocol: %v", err)
	}

	// 2. Create TRP client
	trpClient := tx3.NewTRPClient(trp.ClientOptions{
		Endpoint: endpoint,
	})

	// 3. Create Cardano signer from mnemonic
	cardanoSigner, err := signer.CardanoSignerFromMnemonic(senderAddr, senderMnemonic)
	if err != nil {
		log.Fatalf("Failed to create signer: %v", err)
	}

	// 4. Build client with profile and parties
	client := tx3.NewClient(protocol, trpClient).
		WithProfile("preprod").
		WithParty("sender", tx3.SignerParty(cardanoSigner)).
		WithParty("receiver", tx3.AddressParty(receiverAddr)).
		WithParty("middleman", tx3.AddressParty(receiverAddr))

	ctx := context.Background()

	// 5. Build, resolve, sign, submit, and wait for confirmation
	resolved, err := client.Tx("transfer").
		Arg("quantity", 10_000_000).
		Resolve(ctx)
	if err != nil {
		log.Fatalf("Resolve failed: %v", err)
	}
	fmt.Printf("Resolved tx: %s\n", resolved.Hash)

	signed, err := resolved.Sign()
	if err != nil {
		log.Fatalf("Sign failed: %v", err)
	}
	fmt.Printf("Signed with %d witness(es)\n", len(signed.Witnesses()))

	submitted, err := signed.Submit(ctx)
	if err != nil {
		log.Fatalf("Submit failed: %v", err)
	}
	fmt.Printf("Submitted tx: %s\n", submitted.Hash)

	status, err := submitted.WaitForConfirmed(ctx, tx3.DefaultPollConfig())
	if err != nil {
		log.Fatalf("WaitForConfirmed failed: %v", err)
	}
	fmt.Printf("Transaction confirmed! Stage: %s, Confirmations: %d\n",
		status.Stage, status.Confirmations)
}
