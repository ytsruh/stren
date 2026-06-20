// Package push provides a thin, isolated wrapper around the Web Push
// protocol. It is the only place in the codebase that knows about VAPID,
// webpush-go, or the push service HTTP contract. All other layers depend
// on the small types defined here so that swapping the implementation
// (e.g. for a different push library, or to add new message fields) is
// a single-file change.
//
// The package exposes three primary units:
//
//   - Keys: persistent VAPID keypair, loaded from or generated into a
//     directory on disk. The private key signs every outbound push; the
//     public key is what we hand to the browser via pushManager.subscribe.
//   - Client: per-subscription send. Wraps webpush-go and surfaces the
//     HTTP status code from the push service so callers can prune dead
//     subscriptions on 404/410.
//   - Service: fan-out to every subscription, with bounded concurrency,
//     automatic dead-subscription cleanup, and aggregated results.
package push

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// File names written into dataDir by LoadOrGenerate. Kept as constants
// (not derived) so they are easy to find in `ls /data`.
const (
	privateKeyFile = "vapid_private.pem"
	publicKeyFile  = "vapid_public.pem"
	pemTypeEC      = "EC PRIVATE KEY"
)

// Keys is a loaded VAPID keypair ready to be used by Client.
//
// PrivateKey is the raw ECDSA P-256 key used to sign the VAPID JWT.
// PublicKey is the matching point, used both for signing and (encoded
// via PublicKeyString) to identify this application to the browser.
type Keys struct {
	Private    *ecdsa.PrivateKey
	Public     *ecdsa.PublicKey
	PrivatePEM []byte
	PublicPEM  []byte
}

// LoadOrGenerate returns a VAPID keypair from dataDir. If both the
// private and public key files are present they are loaded and validated
// as a matching pair. If either file is missing a fresh keypair is
// generated and persisted. The function is safe to call on every server
// startup: existing keys are reused so subscriptions survive restarts.
//
// The dataDir is created if it does not exist. Permissions on the
// private key file are tightened to 0600 since it is a signing secret.
func LoadOrGenerate(dataDir string) (*Keys, error) {
	if dataDir == "" {
		return nil, errors.New("push: dataDir is empty")
	}

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("push: create data dir %q: %w", dataDir, err)
	}

	privPath := filepath.Join(dataDir, privateKeyFile)
	pubPath := filepath.Join(dataDir, publicKeyFile)

	privPEM, privErr := os.ReadFile(privPath)
	pubPEM, pubErr := os.ReadFile(pubPath)

	if privErr == nil && pubErr == nil {
		return loadKeys(privPEM, pubPEM)
	}

	keys, err := generateKeys()
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(privPath, keys.PrivatePEM, 0o600); err != nil {
		return nil, fmt.Errorf("push: write private key: %w", err)
	}
	if err := os.WriteFile(pubPath, keys.PublicPEM, 0o644); err != nil {
		return nil, fmt.Errorf("push: write public key: %w", err)
	}

	return keys, nil
}

// loadKeys parses and cross-validates a previously-written key pair.
func loadKeys(privPEM, pubPEM []byte) (*Keys, error) {
	privBlock, _ := pem.Decode(privPEM)
	if privBlock == nil {
		return nil, errors.New("push: invalid private key PEM")
	}
	priv, err := x509.ParseECPrivateKey(privBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("push: parse private key: %w", err)
	}

	pubBlock, _ := pem.Decode(pubPEM)
	if pubBlock == nil {
		return nil, errors.New("push: invalid public key PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("push: parse public key: %w", err)
	}
	ecPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("push: public key is not ECDSA")
	}

	// Reject mismatched pairs. This guards against a corrupted public
	// key file or someone swapping in a foreign public key without the
	// matching private key (which would make the VAPID JWT invalid).
	if priv.PublicKey.Curve != ecPub.Curve {
		return nil, errors.New("push: private and public keys use different curves")
	}
	if priv.PublicKey.X.Cmp(ecPub.X) != 0 || priv.PublicKey.Y.Cmp(ecPub.Y) != 0 {
		return nil, errors.New("push: public key does not match private key")
	}

	return &Keys{
		Private:    priv,
		Public:     ecPub,
		PrivatePEM: privPEM,
		PublicPEM:  pubPEM,
	}, nil
}

// generateKeys produces a new ECDSA P-256 keypair, which is the curve
// required by the VAPID spec (RFC 8292 §2).
func generateKeys() (*Keys, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("push: generate keypair: %w", err)
	}

	privDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("push: marshal private key: %w", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: pemTypeEC, Bytes: privDER})

	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("push: marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	return &Keys{
		Private:    priv,
		Public:     &priv.PublicKey,
		PrivatePEM: privPEM,
		PublicPEM:  pubPEM,
	}, nil
}

// PublicKeyString returns the URL-safe base64 encoding of the public
// key in the uncompressed-point format expected by
// PushManager.subscribe({applicationServerKey}).
func (k *Keys) PublicKeyString() string {
	if k == nil || k.Public == nil {
		return ""
	}
	// MarshalPKIXPublicKey is the safer path, but webpush-go's
	// VAPIDPublicKey field expects the raw uncompressed point
	// (0x04 || X || Y) as base64url-no-pad. We emit that format
	// directly to avoid an extra round of encoding/decoding inside
	// the library.
	curve := k.Public.Curve.Params()
	byteLen := (curve.BitSize + 7) / 8
	out := make([]byte, 1+2*byteLen)
	out[0] = 0x04
	k.Public.X.FillBytes(out[1 : 1+byteLen])
	k.Public.Y.FillBytes(out[1+byteLen : 1+2*byteLen])
	return base64.RawURLEncoding.EncodeToString(out)
}

// PrivateKeyString returns the URL-safe base64 encoding of the
// private key's scalar (the "d" value from RFC 6979). This is the
// format webpush-go's VAPIDPrivateKey field expects — *not* PEM.
// (The PEM form is what we persist to disk so it can be inspected
// with standard tools, but the library speaks the raw scalar.)
func (k *Keys) PrivateKeyString() string {
	if k == nil || k.Private == nil {
		return ""
	}
	curve := k.Private.Curve.Params()
	byteLen := (curve.BitSize + 7) / 8
	out := make([]byte, byteLen)
	k.Private.D.FillBytes(out)
	return base64.RawURLEncoding.EncodeToString(out)
}
