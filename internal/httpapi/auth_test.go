package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func signHS256(secret, payloadJSON string) string {
	enc := base64.RawURLEncoding.EncodeToString
	header := enc([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := enc([]byte(payloadJSON))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(header + "." + payload))
	sig := enc(mac.Sum(nil))
	return header + "." + payload + "." + sig
}

func TestVerifyHS256AcceptsValidToken(t *testing.T) {
	tok := signHS256("topsecret", `{"tenant_id":"t-1","user_id":"u-9"}`)
	claims, err := verifyHS256("topsecret", tok)
	if err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	if claims["tenant_id"] != "t-1" || claims["user_id"] != "u-9" {
		t.Fatalf("claims wrong: %v", claims)
	}
}

func TestVerifyHS256RejectsTamperedSignature(t *testing.T) {
	tok := signHS256("topsecret", `{"tenant_id":"t-1"}`)
	if _, err := verifyHS256("different-secret", tok); err == nil {
		t.Fatal("expected rejection for wrong secret")
	}
}

func TestVerifyHS256RejectsMalformed(t *testing.T) {
	if _, err := verifyHS256("s", "not-a-jwt"); err == nil {
		t.Fatal("expected rejection for malformed token")
	}
}
