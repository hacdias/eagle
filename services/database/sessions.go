package database

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Session represents a login session.
type Session struct {
	ID      string    `gorm:"primaryKey"`
	Expiry  time.Time
	Created time.Time
}

func (d *Database) AddSession(ctx context.Context, session *Session) error {
	return d.db.WithContext(ctx).Create(session).Error
}

func (d *Database) GetSession(ctx context.Context, id string) (*Session, error) {
	var session Session
	err := d.db.WithContext(ctx).First(&session, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &session, err
}

func (d *Database) DeleteSession(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&Session{}, "id = ?", id).Error
}

func (d *Database) GetSessions(ctx context.Context) ([]*Session, error) {
	var sessions []*Session
	err := d.db.WithContext(ctx).Order("created asc").Find(&sessions).Error
	return sessions, err
}

func (d *Database) DeleteAllSessions(ctx context.Context) error {
	return d.db.WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Session{}).Error
}

func (d *Database) DeleteExpiredSessions(ctx context.Context) error {
	return d.db.WithContext(ctx).
		Where("expiry != ? AND expiry < ?", time.Time{}, time.Now()).
		Delete(&Session{}).Error
}
