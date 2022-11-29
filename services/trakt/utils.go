package trakt

import (
	"encoding/hex"
	"math/rand"
	"time"
)

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func randomString(length int) (string, error) {
	b := make([]byte, length)

	_, err := seededRand.Read(b)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
