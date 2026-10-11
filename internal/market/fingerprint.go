package market

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Fingerprint hashes a market's rules text and parsed terms, so a pair notices when either side's rules change
// after matching (FR-027, lld.md rules_fingerprint).
func Fingerprint(rules string, t Terms) string {
	b, _ := json.Marshal(struct {
		R string
		T Terms
	}{rules, t})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
