package httpx

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const claimsKey contextKey = "jwt_claims"

type Claims struct {
	Sub  string `json:"sub"`  // user UUID
	Role string `json:"role"`
	Exp  int64  `json:"exp"`
	Iat  int64  `json:"iat"`
}

func GenerateToken(userID, role, secret string) (string, error) {
	now := time.Now()
	claims := Claims{
		Sub:  userID,
		Role: role,
		Exp:  now.Add(24 * time.Hour).Unix(),
		Iat:  now.Unix(),
	}

	header := base64url(mustJSON(map[string]string{"alg": "HS256", "typ": "JWT"}))
	payload := base64url(mustJSON(claims))
	sigInput := header + "." + payload
	sig := base64url(sign(sigInput, secret))
	return sigInput + "." + sig, nil
}

func JWTMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				Error(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}
			token := strings.TrimPrefix(auth, "Bearer ")
			claims, err := parseToken(token, secret)
			if err != nil {
				Error(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}

func parseToken(token, secret string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, &jwtErr{"invalid token format"}
	}
	sigInput := parts[0] + "." + parts[1]
	expected := base64url(sign(sigInput, secret))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, &jwtErr{"signature mismatch"}
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var c Claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if time.Now().Unix() > c.Exp {
		return nil, &jwtErr{"token expired"}
	}
	return &c, nil
}

func base64url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func sign(input, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return mac.Sum(nil)
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

type jwtErr struct{ msg string }

func (e *jwtErr) Error() string { return e.msg }
