package indieauth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/thoas/go-funk"
)

var (
	codeChallengeMethods = []string{
		"plain", "S256",
	}
)

func IsValidCodeChallengeMethod(ccm string) bool {
	return funk.ContainsString(codeChallengeMethods, ccm)
}

func ValidateCodeChallenge(ccm, cc, ver string) bool {
	switch ccm {
	case "plain":
		return cc == ver
	case "S256":
		s256 := sha256.Sum256([]byte(ver))
		// trim padding
		a := strings.TrimRight(base64.URLEncoding.EncodeToString(s256[:]), "=")
		b := strings.TrimRight(cc, "=")
		return a == b
	default:
		return false
	}
}
