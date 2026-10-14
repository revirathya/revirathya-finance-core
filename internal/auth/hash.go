package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
)

const chOptions = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

var codePattern = regexp.MustCompile(
	fmt.Sprintf(`^[%s]{4}-[%s]{4}$`, chOptions, chOptions),
)

func GenBindCode() (string, error) {
	choices := make([]byte, 8)
	chLen := big.NewInt(int64(len(chOptions)))
	for i := range 8 {
		pt, err := rand.Int(rand.Reader, chLen)
		if err != nil {
			return "", err
		}
		choices[i] = chOptions[pt.Int64()]
	}

	code := string(choices[:4]) + "-" + string(choices[4:])
	return code, nil
}

func HashCode(code string) string {
	codeHashRaw := sha256.Sum256([]byte(code))
	return hex.EncodeToString(codeHashRaw[:])
}

func ValidateCode(code string) bool {
	return codePattern.MatchString(code)
}
