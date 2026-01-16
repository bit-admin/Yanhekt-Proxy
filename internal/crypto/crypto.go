package crypto

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Crypto struct {
	magicKey string
}

func New(magicKey string) *Crypto {
	return &Crypto{magicKey: magicKey}
}

// EncryptURL inserts MD5 hash before the last path segment
// Example: https://cvideo.yanhekt.cn/path/to/file.ts
//       -> https://cvideo.yanhekt.cn/path/to/<hash>/file.ts
func (c *Crypto) EncryptURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return url
	}

	hash := c.md5Hash(c.magicKey + "_100")

	// Insert hash before the last segment
	lastIdx := len(parts) - 1
	result := make([]string, 0, len(parts)+1)
	result = append(result, parts[:lastIdx]...)
	result = append(result, hash)
	result = append(result, parts[lastIdx])

	return strings.Join(result, "/")
}

// GetSignature generates timestamp and MD5 signature
func (c *Crypto) GetSignature() (timestamp, signature string) {
	timestamp = fmt.Sprintf("%d", time.Now().Unix())
	signature = c.md5Hash(c.magicKey + "_v1_" + timestamp)
	return
}

// SignURL appends all authentication parameters to URL
func (c *Crypto) SignURL(url, videoToken string) string {
	timestamp, signature := c.GetSignature()
	return fmt.Sprintf("%s?Xvideo_Token=%s&Xclient_Timestamp=%s&Xclient_Signature=%s&Xclient_Version=v1&Platform=yhkt_user",
		url, videoToken, timestamp, signature)
}

func (c *Crypto) md5Hash(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}
