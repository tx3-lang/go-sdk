//go:build codegen

// This test exercises the .trix/client-lib codegen plugin (Handlebars
// templates + tx3c), not the SDK runtime. It is gated behind the `codegen`
// build tag so it runs only in the dedicated codegen CI job.
package tx3sdk_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// resolveTx3c locates the tx3c binary: TX3_TX3C_PATH first, then $PATH.
func resolveTx3c(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("TX3_TX3C_PATH"); p != "" {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	if p, err := exec.LookPath("tx3c"); err == nil {
		return p
	}
	t.Fatal("tx3c not found; set TX3_TX3C_PATH or install tx3c")
	return ""
}

// TestCodegenClientLib renders the .trix/client-lib plugin against the shared
// transfer fixture, asserts the expected public surface is present, and builds
// the result. A successful render that produces uncompilable or empty bindings
// is a failure.
func TestCodegenClientLib(t *testing.T) {
	sdkDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoRoot := filepath.Dir(sdkDir)
	templateDir := filepath.Join(repoRoot, ".trix", "client-lib")
	tiiPath := filepath.Join(sdkDir, "testdata", "transfer.tii")

	if _, err := os.Stat(tiiPath); err != nil {
		t.Fatalf("missing TII fixture: %v", err)
	}
	if _, err := os.Stat(templateDir); err != nil {
		t.Fatalf("missing template directory: %v", err)
	}

	outDir := t.TempDir()

	render := exec.Command(resolveTx3c(t), "codegen",
		"--tii", tiiPath,
		"--template", templateDir,
		"--output", outDir,
	)
	if out, err := render.CombinedOutput(); err != nil {
		t.Fatalf("tx3c codegen failed: %v\n%s", err, out)
	}

	for _, name := range []string{"protocol.go", "go.mod"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Fatalf("expected generated file %s: %v", name, err)
		}
	}

	// Smoke-test the generated surface: the template must emit protocol
	// identity, the per-transaction types, and the profile surface.
	src, err := os.ReadFile(filepath.Join(outDir, "protocol.go"))
	if err != nil {
		t.Fatalf("read generated protocol.go: %v", err)
	}
	for _, want := range []string{
		"TargetTIIVersion",
		"type TransferParams struct",
		"TRANSFER_TIR",
		"func (c *Client) Transfer(",
		"var Profiles",
	} {
		if !strings.Contains(string(src), want) {
			t.Errorf("generated protocol.go missing expected symbol: %s", want)
		}
	}

	// Build the rendered package against this repo's SDK rather than a
	// published release.
	goMod := "module codegentest\n\ngo 1.24.2\n\n" +
		"require github.com/tx3-lang/go-sdk/sdk v0.0.0\n\n" +
		"replace github.com/tx3-lang/go-sdk/sdk => " + sdkDir + "\n"
	if err := os.WriteFile(filepath.Join(outDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if sum, err := os.ReadFile(filepath.Join(sdkDir, "go.sum")); err == nil {
		if err := os.WriteFile(filepath.Join(outDir, "go.sum"), sum, 0o644); err != nil {
			t.Fatalf("write go.sum: %v", err)
		}
	}

	build := exec.Command("go", "build", "./...")
	build.Dir = outDir
	build.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build of generated package failed: %v\n%s", err, out)
	}
}
