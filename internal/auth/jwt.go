package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	TokenAccess  = "access"
	TokenRefresh = "refresh"
)

// TokenManager issues and verifies HS256 JWTs for parent accounts.
type TokenManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenManager(secret string) *TokenManager {
	return &TokenManager{
		secret:     []byte(secret),
		accessTTL:  15 * time.Minute,
		refreshTTL: 30 * 24 * time.Hour,
	}
}

func (tm *TokenManager) AccessTTL() time.Duration { return tm.accessTTL }

type claims struct {
	jwt.RegisteredClaims
	TokenType string `json:"typ"`
}

func (tm *TokenManager) issue(accountID uuid.UUID, ttl time.Duration, typ string) (string, error) {
	now := time.Now()
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   accountID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		TokenType: typ,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(tm.secret)
}

func (tm *TokenManager) AccessToken(accountID uuid.UUID) (string, error) {
	return tm.issue(accountID, tm.accessTTL, TokenAccess)
}

func (tm *TokenManager) RefreshToken(accountID uuid.UUID) (string, error) {
	return tm.issue(accountID, tm.refreshTTL, TokenRefresh)
}

// Parse verifies the token's signature and type, returning the account id.
func (tm *TokenManager) Parse(token, expectedType string) (uuid.UUID, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return tm.secret, nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	if c.TokenType != expectedType {
		return uuid.Nil, fmt.Errorf("expected %s token, got %q", expectedType, c.TokenType)
	}
	return uuid.Parse(c.Subject)
}
