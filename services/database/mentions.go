package database

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.hacdias.com/eagle/core"
	"gorm.io/gorm"
)

func (d *Database) AddMention(ctx context.Context, mention *core.Mention) error {
	mention.ID = uuid.New().String()
	return d.db.WithContext(ctx).Create(mention).Error
}

func (d *Database) GetMentions(ctx context.Context) ([]*core.Mention, error) {
	var mentions []*core.Mention
	err := d.db.WithContext(ctx).Find(&mentions).Error
	return mentions, err
}

func (d *Database) GetMention(ctx context.Context, id string) (*core.Mention, error) {
	var mention core.Mention
	err := d.db.WithContext(ctx).First(&mention, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("mention does not exist")
	}
	return &mention, err
}

func (d *Database) DeleteMention(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&core.Mention{}, "id = ?", id).Error
}
