package database

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Token represents an IndieAuth JWT token persisted in the database.
type Token struct {
	ID       string `gorm:"primaryKey"`
	ClientID string
	Scope    string
	Expiry   time.Time // zero means no expiry
	Created  time.Time
}

func (d *Database) AddToken(ctx context.Context, token *Token) error {
	return d.db.WithContext(ctx).Create(token).Error
}

func (d *Database) HasToken(ctx context.Context, id string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&Token{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

func (d *Database) DeleteToken(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&Token{}, "id = ?", id).Error
}

func (d *Database) GetTokens(ctx context.Context) ([]*Token, error) {
	var tokens []*Token
	err := d.db.WithContext(ctx).Order("created asc").Find(&tokens).Error
	return tokens, err
}

func (d *Database) DeleteExpiredTokens(ctx context.Context) error {
	return d.db.WithContext(ctx).
		Where("expiry != ? AND expiry < ?", time.Time{}, time.Now()).
		Delete(&Token{}).Error
}

func (d *Database) DeleteAllTokens(ctx context.Context) error {
	return d.db.WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Token{}).Error
}
