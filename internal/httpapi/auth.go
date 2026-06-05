package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type ctxKey string

const (
	ctxTenant ctxKey = "tenant_id"
	ctxUser   ctxKey = "user_id"
)

// identity pulls the (optional) tenant_id and user_id placed by authMiddleware.
func identity(ctx context.Context) (tenantID, userID *string) {
	if v, ok := ctx.Value(ctxTenant).(string); ok && v != "" {
		tenantID = &v
	}
	if v, ok := ctx.Value(ctxUser).(string); ok && v != "" {
		userID = &v
	}
	return
}

// authMiddleware resolves the caller's tenant_id/user_id. If secret is set it
// validates a Bearer HS256 JWT and reads the claims; otherwise it runs in
// permissive (dev) mode, reading X-Tenant-Id / X-User-Id headers. Wired for
// production, non-blocking locally (PRD Phase 1 decision: optional/stub auth).
func authMiddleware(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tenant, user string
		if secret == "" {
			tenant = r.Header.Get("X-Tenant-Id")
			user = r.Header.Get("X-User-Id")
		} else {
			claims, err := verifyHS256(secret, bearer(r))
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			tenant, _ = claims["tenant_id"].(string)
			if user, _ = claims["user_id"].(string); user == "" {
				user, _ = claims["sub"].(string)
			}
		}
		ctx := context.WithValue(r.Context(), ctxTenant, tenant)
		ctx = context.WithValue(ctx, ctxUser, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[len("Bearer "):])
	}
	return ""
}

// verifyHS256 validates a compact JWS (HS256) signature and returns its claims.
// Minimal, dependency-free; covers signature + alg only (exp/nbf checks would be
// added when wired to the real Apolaki key).
func verifyHS256(secret, token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token")
	}
	var header struct {
		Alg string `json:"alg"`
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("bad header: %w", err)
	}
	if err := json.Unmarshal(hb, &header); err != nil {
		return nil, fmt.Errorf("bad header json: %w", err)
	}
	if header.Alg != "HS256" {
		return nil, fmt.Errorf("unsupported alg %q", header.Alg)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, fmt.Errorf("bad signature")
	}
	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("bad payload: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(pb, &claims); err != nil {
		return nil, fmt.Errorf("bad payload json: %w", err)
	}
	return claims, nil
}
