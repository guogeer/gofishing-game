package cache

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"gofishing-game/internal/env"
)

var tokenKey = "lolbye2023" + env.Config().Sign

func GenToken(uid int) string {
	sign := fmt.Sprintf("%s_%d", tokenKey, uid)
	sum := md5.Sum([]byte(sign))
	hexSum := hex.EncodeToString(sum[:])
	return hexSum
}
