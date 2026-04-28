package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
)

const AccessTokenCookieName = "rg_access_token"

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type JWTClaims struct {
	Subject  string `json:"sub"`
	Username string `json:"username"`
	IssuedAt int64  `json:"iat"`
	Expires  int64  `json:"exp"`
}

func IssueToken(userID, username, secret string, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("jwt secret is required")
	}
	if ttl <= 0 {
		return "", errors.New("jwt ttl must be positive")
	}

	headerJSON, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}

	now := time.Now().Unix()
	claimsJSON, err := json.Marshal(JWTClaims{
		Subject:  userID,
		Username: username,
		IssuedAt: now,
		Expires:  now + int64(ttl.Seconds()),
	})
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsigned := encodedHeader + "." + encodedClaims

	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + signature, nil
}

func ParseToken(token, secret string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}
	expectedSignature := mac.Sum(nil)

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}
	if !hmac.Equal(signature, expectedSignature) {
		return nil, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims JWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrInvalidToken
	}
	if claims.Subject == "" || claims.Expires == 0 {
		return nil, ErrInvalidToken
	}
	if time.Now().Unix() >= claims.Expires {
		return nil, ErrExpiredToken
	}

	return &claims, nil
}

func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(AccessTokenCookieName)
		if err != nil || token == "" {
			response.Error(c, 40100, "unauthorized")
			c.Abort()
			return
		}

		claims, err := ParseToken(token, secret)
		if err != nil {
			response.Error(c, 40100, "unauthorized")
			c.Abort()
			return
		}

		c.Set(ContextUserID, claims.Subject)
		c.Set("username", claims.Username)
		c.Next()
	}
}
