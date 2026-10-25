package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckPublicAddress(t *testing.T) {
	for _, address := range []string{
		"127.0.0.1:80",
		"0.0.0.0:80",
		"169.254.169.254:80",
		"10.0.0.1:80",
		"192.168.1.1:80",
		"[::1]:80",
		"[::]:80",
		"[fe80::1]:80",
		"[fd00::1]:80",
		"[::ffff:127.0.0.1]:80",
	} {
		assert.Error(t, checkPublicAddress("tcp", address, nil), address)
	}

	for _, address := range []string{
		"93.184.216.34:443",
		"[2606:2800:220:1:248:1893:25c8:1946]:443",
	} {
		assert.NoError(t, checkPublicAddress("tcp", address, nil), address)
	}
}
