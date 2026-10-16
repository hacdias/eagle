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
	Expiry   time.Time
	Created  time.Time
}
