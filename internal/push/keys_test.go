package push

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validKeys returns a freshly-generated keypair, mirroring what
// LoadOrGenerate would write to disk in production. The point of the
// test fixture is to have a known-good pair we can round-trip without
// pulling in test data files.
func validKeys(t *testing.T) *Keys {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	privDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal priv: %v", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: pemTypeEC, Bytes: privDER})

	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("marshal pub: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	return &Keys{
		Private:    priv,
		Public:     &priv.PublicKey,
		PrivatePEM: privPEM,
		PublicPEM:  pubPEM,
	}
}

func TestLoadOrGenerate_FreshDir_GeneratesPair(t *testing.T) {
	dir := t.TempDir()
	k, err := LoadOrGenerate(dir)
	if err != nil {
		t.Fatalf("LoadOrGenerate: %v", err)
	}
	if k.Private == nil || k.Public == nil {
		t.Fatal("expected non-nil keys")
	}
	// Files should be on disk and the same ones we just generated.
	if _, err := os.Stat(filepath.Join(dir, privateKeyFile)); err != nil {
		t.Fatalf("private key file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, publicKeyFile)); err != nil {
		t.Fatalf("public key file: %v", err)
	}
}

func TestLoadOrGenerate_ExistingDir_ReusesPair(t *testing.T) {
	dir := t.TempDir()
	first, err := LoadOrGenerate(dir)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}

	second, err := LoadOrGenerate(dir)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}

	// The keys should be byte-for-byte the same on disk reload.
	if first.Private.D.Cmp(second.Private.D) != 0 {
		t.Fatal("expected same private key on reload")
	}
	if first.Public.X.Cmp(second.Public.X) != 0 {
		t.Fatal("expected same public key on reload")
	}
}

func TestLoadOrGenerate_RejectsMismatchedPair(t *testing.T) {
	dir := t.TempDir()
	good := validKeys(t)
	other := validKeys(t)

	// Write a private key with a *different* public key alongside it.
	if err := os.WriteFile(filepath.Join(dir, privateKeyFile), good.PrivatePEM, 0o600); err != nil {
		t.Fatalf("write priv: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, publicKeyFile), other.PublicPEM, 0o644); err != nil {
		t.Fatalf("write pub: %v", err)
	}

	if _, err := LoadOrGenerate(dir); err == nil {
		t.Fatal("expected error on mismatched keypair")
	}
}

func TestLoadOrGenerate_RejectsInvalidPEM(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, privateKeyFile), []byte("not pem"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, publicKeyFile), []byte("not pem"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadOrGenerate(dir); err == nil {
		t.Fatal("expected error on invalid PEM")
	}
}

func TestLoadOrGenerate_RejectsEmptyDataDir(t *testing.T) {
	if _, err := LoadOrGenerate(""); err == nil {
		t.Fatal("expected error for empty dataDir")
	}
}

func TestKeys_PublicKeyString(t *testing.T) {
	k := validKeys(t)
	s := k.PublicKeyString()
	if s == "" {
		t.Fatal("expected non-empty string")
	}
	// Must be base64url-no-pad (no '+', '/', or '=' in output).
	if strings.ContainsAny(s, "+/=") {
		t.Fatalf("expected base64url-no-pad, got %q", s)
	}
}

func TestKeys_PublicKeyString_NilSafe(t *testing.T) {
	var k *Keys
	if s := k.PublicKeyString(); s != "" {
		t.Fatalf("expected empty string for nil keys, got %q", s)
	}
}
