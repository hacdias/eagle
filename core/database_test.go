package core

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestConsumeAuthCode(t *testing.T) {
	_, db := newTestQueue(t)
	ctx := context.Background()

	id := HashToken("code")
	require.NoError(t, db.CreateAuthCode(ctx, &AuthCode{
		ID:       id,
		ClientID: "https://client.example/",
		Expiry:   time.Now().Add(time.Minute),
	}))

	code, err := db.ConsumeAuthCode(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "https://client.example/", code.ClientID)

	_, err = db.ConsumeAuthCode(ctx, id)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestGetTokenByHash(t *testing.T) {
	_, db := newTestQueue(t)
	ctx := context.Background()

	require.NoError(t, db.CreateToken(ctx, &Token{
		ID:     HashToken("raw"),
		Type:   TokenTypeAccess,
		Expiry: time.Now().Add(time.Minute),
	}))

	_, err := db.GetToken(ctx, HashToken("raw"), TokenTypeAccess)
	assert.NoError(t, err)

	_, err = db.GetToken(ctx, "raw", TokenTypeAccess)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
