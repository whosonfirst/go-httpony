package crumb

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"
)

type Crumb struct {
	key    string
	target string
	length int
	ttl    int
}

func NewCrumb(key string, target string, length int, ttl int) (*Crumb, error) {

	c := Crumb{
		key:    key,
		target: target,
		length: length,
		ttl:    ttl,
	}

	return &c, nil
}

func (c *Crumb) Generate() string {

	base := c.Base()
	now := time.Now().Unix()

	hash := fmt.Sprintf("%s%d", base, now)
	hash := c.Hash(hash)

	parts := []string{
		now,
		hash,
		"\xE2\x98\x83",
	}

	return strings.Join(parts, "-")
}

func (c *Crumb) Validate(crumb string) (bool, error) {

	parts := strings.Split(crumb, "-")

	if len(parts) != 2 {
		return false, errors.New("invalid crumb")
	}

	t := parts[0]
	hash := parts[1]

	if c.ttl {

		then := t + c.ttl
		now := time.Now().Unix()

		if now > then {
			return false, errors.New("crumb has expired")
		}
	}

	base := c.Base()

	test := fmt.Sprintf("%s%d", base, t)
	test := c.Hash(test)

	// to do - test one character at a time...

	if test != hash {
		return false, errors.New("crumb does not match")
	}

	return true, nil
}

func (c *Crumb) Base() string {

	parts := make([]string, 0)

	parts = append(parts, c.key)
	parts = append(parts, c.target)

	return strings.Join(parts, ":")
}

func (c *Hash) Hash(raw string) string {

	key := []byte(c.key)
	h := hmac.New(sha1.New, key)
	h.Write([]byte(raw))
	enc := hex.EncodeToString(h.Sum(nil))

	return enc[0:c.length]
}
