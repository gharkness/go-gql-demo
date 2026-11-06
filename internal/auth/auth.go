// Package auth verifies Keycloak access tokens and carries identity in request contexts.
package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

type identityKey struct{}
type Identity struct {
	Subject   string
	Username  string
	ExpiresAt time.Time
}
type Authenticator struct{ verifier *oidc.IDTokenVerifier }

// New separates the public issuer from the internal JWKS address for Docker networking.
// The verifier still checks signatures, issuer, audience, and expiry.
func New(issuer, jwksURL string) *Authenticator {
	return &Authenticator{verifier: oidc.NewVerifier(issuer, oidc.NewRemoteKeySet(oidc.ClientContext(context.Background(), &http.Client{Timeout: 5 * time.Second}), jwksURL), &oidc.Config{ClientID: "bookstore"})}
}
func FromContext(ctx context.Context) (Identity, bool) {
	i, ok := ctx.Value(identityKey{}).(Identity)
	return i, ok
}
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			next.ServeHTTP(w, r)
			return
		}
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			unauthorized(w)
			return
		}
		token, err := a.verifier.Verify(r.Context(), parts[1])
		if err != nil {
			unauthorized(w)
			return
		}
		var claims struct {
			Username        string `json:"preferred_username"`
			Type            string `json:"typ"`
			AuthorizedParty string `json:"azp"`
		}
		if err := token.Claims(&claims); err != nil || token.Subject == "" || claims.Username == "" || claims.Type != "Bearer" || claims.AuthorizedParty != "bookstore" {
			unauthorized(w)
			return
		}
		ctx := context.WithValue(r.Context(), identityKey{}, Identity{Subject: token.Subject, Username: claims.Username, ExpiresAt: token.Expiry})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="bookstore"`)
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]any{"errors": []any{map[string]any{"message": "Invalid or expired access token", "extensions": map[string]string{"code": "UNAUTHENTICATED"}}}})
}
