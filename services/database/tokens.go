package database

import (
	"context"
	"time"

	"go.hacdias.com/eagle/core"
)

func (d *Database) CreateToken(ctx context.Context, token *core.Token) error {
	return d.db.WithContext(ctx).Create(token).Error
}

func (d *Database) GetToken(ctx context.Context, id string, tokenType core.TokenType) (*core.Token, error) {
	var token core.Token
	err := d.db.WithContext(ctx).First(&token, "id = ? AND type = ?", id, tokenType).Error
	return &token, err
}

func (d *Database) GetTokensByType(ctx context.Context, tokenType core.TokenType) ([]*core.Token, error) {
	var tokens []*core.Token
	err := d.db.WithContext(ctx).Where("type = ?", tokenType).Order("created asc").Find(&tokens).Error
	return tokens, err
}

func (d *Database) DeleteToken(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&core.Token{}, "id = ?", id).Error
}

func (d *Database) DeleteExpiredTokens(ctx context.Context) error {
	return d.db.WithContext(ctx).
		Where("expiry != ? AND expiry < ?", time.Time{}, time.Now()).
		Delete(&core.Token{}).Error
}

func (d *Database) DeleteAllTokensByType(ctx context.Context, tokenType core.TokenType) error {
	return d.db.WithContext(ctx).Where("type = ?", tokenType).Delete(&core.Token{}).Error
}
