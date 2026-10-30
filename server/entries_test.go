package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.hacdias.com/eagle/core"
	"gorm.io/gorm"
)

func TestUpdateSchedule(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	db := s.core.DB()

	e := &core.Entry{
		ID: "/posts/2026/01/01/post/",
		FrontMatter: core.FrontMatter{
			Date: time.Now().Add(time.Hour),
		},
	}

	// Future entries are scheduled instead of syndicated.
	syndicators, status := s.updateSchedule(e, []string{"mastodon"}, "Hello")
	assert.Empty(t, syndicators)
	assert.Empty(t, status)

	scheduled, err := db.GetScheduledEntry(ctx, e.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"mastodon"}, scheduled.Syndicators)
	assert.Equal(t, "Hello", scheduled.SyndicationStatus)

	// Once published, the scheduled syndicators and status are used, and the
	// schedule is removed.
	e.Date = time.Now().Add(-time.Minute)
	syndicators, status = s.updateSchedule(e, []string{"atproto"}, "")
	assert.ElementsMatch(t, []string{"atproto", "mastodon"}, syndicators)
	assert.Equal(t, "Hello", status)

	_, err = db.GetScheduledEntry(ctx, e.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Drafts are not syndicated, and their schedule is removed.
	e.Date = time.Now().Add(time.Hour)
	s.updateSchedule(e, []string{"mastodon"}, "")

	e.Draft = true
	syndicators, _ = s.updateSchedule(e, []string{"atproto"}, "")
	assert.Empty(t, syndicators)

	_, err = db.GetScheduledEntry(ctx, e.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
