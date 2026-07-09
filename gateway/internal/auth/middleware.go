/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package auth provides HTTP middleware for validating ****** sent by
// external clients (e.g. a Pterodactyl panel) to the ptero-wings-gateway.
//
// Token validation:
//   - A plain static token is compared via constant-time equality when JWT is disabled.
//   - When JWT is enabled, the token is a signed HS256 JWT; the signature is verified
//     against the configured secret and the exp/nbf claims are checked.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Config holds the auth configuration.
type Config struct {
	// Token is the expected static bearer token or the HMAC-SHA256 secret for JWT validation.
	Token string
	// UseJWT switches to JWT HS256 validation mode when true.
	UseJWT bool
}

// Middleware returns an http.Handler that wraps next with bearer-token authentication.
func Middleware(cfg Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearer(r)
		if token == "" {
			http.Error(w, `{"error":"missing Authorization header"}`, http.StatusUnauthorized)
			return
		}

		var ok bool
		var err error
		if cfg.UseJWT {
			ok, err = validateJWT(token, cfg.Token)
		} else {
			ok = hmac.Equal([]byte(token), []byte(cfg.Token))
		}

		if err != nil || !ok {
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusUnauthorized)
				return
			}
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractBearer parses the Authorization header and returns the bearer token value.
func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

// jwtClaims holds the fields we verify in a signed JWT.
type jwtClaims struct {
	Exp int64 `json:"exp"`
	Nbf int64 `json:"nbf"`
}

// validateJWT verifies an HS256-signed JWT against secret and checks exp/nbf.
// It does NOT validate the iss/aud claims — callers should add those as needed.
func validateJWT(tokenStr, secret string) (bool, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return false, fmt.Errorf("malformed JWT")
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return false, fmt.Errorf("invalid JWT signature")
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false, fmt.Errorf("decoding JWT payload: %w", err)
	}
	var claims jwtClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return false, fmt.Errorf("parsing JWT claims: %w", err)
	}

	now := time.Now().Unix()
	if claims.Exp > 0 && now > claims.Exp {
		return false, fmt.Errorf("JWT expired")
	}
	if claims.Nbf > 0 && now < claims.Nbf {
		return false, fmt.Errorf("JWT not yet valid")
	}

	return true, nil
}
