package tii_test

import (
	"errors"
	"os"
	"testing"

	"github.com/tx3-lang/go-sdk/sdk/tii"
)

func TestProtocolFromFile(t *testing.T) {
	p, err := tii.FromFile("../testdata/transfer.tii")
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}
	assertProtocolValid(t, p)
}

func TestProtocolFromString(t *testing.T) {
	data, err := os.ReadFile("../testdata/transfer.tii")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	p, err := tii.FromString(string(data))
	if err != nil {
		t.Fatalf("FromString failed: %v", err)
	}
	assertProtocolValid(t, p)
}

func TestProtocolFromBytes(t *testing.T) {
	data, err := os.ReadFile("../testdata/transfer.tii")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	p, err := tii.FromBytes(data)
	if err != nil {
		t.Fatalf("FromBytes failed: %v", err)
	}
	assertProtocolValid(t, p)
}

func TestFromFileAndFromStringProduceEquivalentProtocol(t *testing.T) {
	pFile, err := tii.FromFile("../testdata/transfer.tii")
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	data, _ := os.ReadFile("../testdata/transfer.tii")
	pStr, err := tii.FromString(string(data))
	if err != nil {
		t.Fatalf("FromString failed: %v", err)
	}

	// Both should expose the same transactions and parties
	if len(pFile.Transactions()) != len(pStr.Transactions()) {
		t.Errorf("transaction count mismatch: file=%d, str=%d",
			len(pFile.Transactions()), len(pStr.Transactions()))
	}
	if len(pFile.Parties()) != len(pStr.Parties()) {
		t.Errorf("party count mismatch: file=%d, str=%d",
			len(pFile.Parties()), len(pStr.Parties()))
	}
}

func TestRejectMalformedJSON(t *testing.T) {
	_, err := tii.FromString(`{invalid json`)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	var jsonErr *tii.InvalidJSONError
	if !errors.As(err, &jsonErr) {
		t.Fatalf("expected InvalidJSONError, got %T: %v", err, err)
	}
}

func TestRejectMissingFile(t *testing.T) {
	_, err := tii.FromFile("nonexistent.tii")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	var ioErr *tii.IOError
	if !errors.As(err, &ioErr) {
		t.Fatalf("expected IOError, got %T: %v", err, err)
	}
}

func TestUnknownTransactionName(t *testing.T) {
	p, err := tii.FromFile("../testdata/transfer.tii")
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	_, err = p.Invoke("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown transaction")
	}
	var txErr *tii.UnknownTxError
	if !errors.As(err, &txErr) {
		t.Fatalf("expected UnknownTxError, got %T: %v", err, err)
	}
	if txErr.Name != "nonexistent" {
		t.Errorf("expected name 'nonexistent', got %q", txErr.Name)
	}
}

func TestUnknownProfile(t *testing.T) {
	p, err := tii.FromFile("../testdata/transfer.tii")
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	profile := "nonexistent"
	_, err = p.Invoke("transfer", &profile)
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
	var profErr *tii.UnknownProfileError
	if !errors.As(err, &profErr) {
		t.Fatalf("expected UnknownProfileError, got %T: %v", err, err)
	}
}

func TestInvokeWithProfile(t *testing.T) {
	p, err := tii.FromFile("../testdata/transfer.tii")
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	profile := "preprod"
	inv, err := p.Invoke("transfer", &profile)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	// Preprod profile sets tax=5000000, so "quantity" should be the only unspecified param
	unspecified := inv.UnspecifiedParams()
	for _, name := range unspecified {
		if name == "tax" {
			t.Error("tax should be populated from preprod profile")
		}
	}
}

// TestInvokeInterpretsComplexParams locks in the Protocol.Invoke path that the
// unit tests can't reach: threading spec.Components into ParamTypeFromSchema,
// and exposing party (Address) and environment-schema params. Asserts a real
// complex.tii produces the expected compound kinds, incl. a component-$ref Record.
func TestInvokeInterpretsComplexParams(t *testing.T) {
	p, err := tii.FromFile("../testdata/complex.tii")
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	inv, err := p.Invoke("complex", nil)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	params := inv.Params()

	wantKind := map[string]tii.ParamKind{
		"quantity":  tii.KindInteger,
		"flag":      tii.KindBoolean,
		"nothing":   tii.KindUnit,
		"recipient": tii.KindAddress,
		"source":    tii.KindUtxoRef,
		"bag":       tii.KindAnyAsset,
		"amounts":   tii.KindList,
		"pair":      tii.KindTuple,
		"labels":    tii.KindMap,
		"asset":     tii.KindRecord,
		"side":      tii.KindVariant,
		// Parties surface as implicit Address params (lowercased).
		"sender":   tii.KindAddress,
		"receiver": tii.KindAddress,
		// Protocol-level environment schema params.
		"fee": tii.KindInteger,
	}
	for name, want := range wantKind {
		got, ok := params[name]
		if !ok {
			t.Errorf("missing param %q", name)
			continue
		}
		if got.Kind != want {
			t.Errorf("param %q: got kind %d, want %d", name, got.Kind, want)
		}
	}

	// The component-$ref Record must have resolved its inner Bytes field — this
	// is the assertion that actually guards the spec.Components threading.
	asset := params["asset"]
	if asset.Fields["policy"].Kind != tii.KindBytes {
		t.Errorf("asset.policy: got kind %d, want Bytes", asset.Fields["policy"].Kind)
	}

	// The component-$ref Variant must have resolved its cases.
	if side := params["side"]; len(side.Cases) != 2 {
		t.Errorf("side: got %d cases, want 2", len(side.Cases))
	}
}

func assertProtocolValid(t *testing.T, p *tii.Protocol) {
	t.Helper()

	txs := p.Transactions()
	if _, ok := txs["transfer"]; !ok {
		t.Error("expected 'transfer' transaction")
	}

	parties := p.Parties()
	expectedParties := []string{"sender", "receiver", "middleman"}
	for _, name := range expectedParties {
		if _, ok := parties[name]; !ok {
			t.Errorf("expected party %q", name)
		}
	}

	profiles := p.Profiles()
	if _, ok := profiles["preprod"]; !ok {
		t.Error("expected 'preprod' profile")
	}

	if p.TiiVersion() != "v1beta0" {
		t.Errorf("expected TII version v1beta0, got %s", p.TiiVersion())
	}
}
