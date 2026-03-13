package core

import "time"

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
	Expiry   time.Time // zero means no expiry, only for access tokens
	Created  time.Time
}
