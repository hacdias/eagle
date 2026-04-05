package core

import (
	"context"
	"time"

	"go.hacdias.com/eagle/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database struct {
	db *gorm.DB
}

func newDatabase(path string) (*Database, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: log.NewGormLogger(),
	})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Token{}, &Mention{}, &QueueItem{})
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Token methods

func (d *Database) CreateToken(ctx context.Context, token *Token) error {
	return d.db.WithContext(ctx).Create(token).Error
}

func (d *Database) GetToken(ctx context.Context, id string, tokenType TokenType) (*Token, error) {
	var token Token
	err := d.db.WithContext(ctx).First(&token, "id = ? AND type = ?", id, tokenType).Error
	return &token, err
}

func (d *Database) GetTokensByType(ctx context.Context, tokenType TokenType) ([]*Token, error) {
	var tokens []*Token
	err := d.db.WithContext(ctx).Where("type = ?", tokenType).Order("created asc").Find(&tokens).Error
	return tokens, err
}

func (d *Database) DeleteToken(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&Token{}, "id = ?", id).Error
}

func (d *Database) DeleteExpiredTokens(ctx context.Context) error {
	return d.db.WithContext(ctx).
		Where("expiry != ? AND expiry < ?", time.Time{}, time.Now()).
		Delete(&Token{}).Error
}

func (d *Database) DeleteAllTokensByType(ctx context.Context, tokenType TokenType) error {
	return d.db.WithContext(ctx).Where("type = ?", tokenType).Delete(&Token{}).Error
}

// Mention methods

func (d *Database) CreateMention(ctx context.Context, mention *Mention) error {
	return d.db.WithContext(ctx).Create(mention).Error
}

func (d *Database) GetMention(ctx context.Context, id string) (*Mention, error) {
	var mention Mention
	err := d.db.WithContext(ctx).First(&mention, "id = ?", id).Error
	return &mention, err
}

func (d *Database) GetMentions(ctx context.Context) ([]*Mention, error) {
	var mentions []*Mention
	err := d.db.WithContext(ctx).Find(&mentions).Error
	return mentions, err
}

func (d *Database) GetMentionBySourceAndEntry(ctx context.Context, source, entryID string) (*Mention, error) {
	var mention Mention
	err := d.db.WithContext(ctx).First(&mention, "source = ? AND entry_id = ?", source, entryID).Error
	return &mention, err
}

func (d *Database) UpdateMention(ctx context.Context, mention *Mention) error {
	return d.db.WithContext(ctx).Save(mention).Error
}

func (d *Database) DeleteMention(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&Mention{}, "id = ?", id).Error
}

// Queue methods

func (d *Database) CreateQueueItem(ctx context.Context, item *QueueItem) error {
	return d.db.WithContext(ctx).Create(item).Error
}

func (d *Database) GetPendingQueueItems(ctx context.Context, n, maxAttempts int, retryAfter time.Time) ([]*QueueItem, error) {
	var items []*QueueItem
	err := d.db.WithContext(ctx).
		Where("attempts < ? AND (last_attempt IS NULL OR last_attempt <= ?)", maxAttempts, retryAfter).
		Order("created asc").
		Limit(n).
		Find(&items).Error
	return items, err
}

func (d *Database) UpdateQueueItem(ctx context.Context, item *QueueItem) error {
	return d.db.WithContext(ctx).Save(item).Error
}

func (d *Database) DeleteQueueItem(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&QueueItem{}, "id = ?", id).Error
}

func (d *Database) GetFailedQueueItems(ctx context.Context) ([]*QueueItem, error) {
	var items []*QueueItem
	err := d.db.WithContext(ctx).Where("attempts >= ?", 3).Order("created asc").Find(&items).Error
	return items, err
}

func (d *Database) GetActiveQueueItems(ctx context.Context) ([]*QueueItem, error) {
	var items []*QueueItem
	err := d.db.WithContext(ctx).Where("attempts < ?", 3).Order("created asc").Find(&items).Error
	return items, err
}

func (d *Database) DeleteFailedQueueItems(ctx context.Context) error {
	return d.db.WithContext(ctx).Where("attempts >= ?", 3).Delete(&QueueItem{}).Error
}
