package util

import (
	"math/rand"
	"strconv"
	"time"
)

// RndDigigtsStr returns random string with digits
func RndDigitsStr(len int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var res int32
	for i := 0; i < len; i++ {
		digit := r.Int31n(10)
		res = res*10 + digit
	}
	return strconv.FormatInt(int64(res), 10)
}
