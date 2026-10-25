package core

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type TokenType string

const (
	TokenTypeSession TokenType = "session"
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Token struct {
	ID       string
	Type     TokenType `gorm:"index"`
	ClientID string
	Scope    string
	Expiry   time.Time
	Created  time.Time
}

type AuthCode struct {
	ID                  string
	ClientID            string
	RedirectURI         string
	Scope               string
	CodeChallenge       string
	CodeChallengeMethod string
	Lifetime            time.Duration // lifetime of the access token to issue
	Expiry              time.Time
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
