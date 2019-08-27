package utils

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

func GenCode(width int) string {
	numeric := [10]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	r := len(numeric)
	rand.Seed(time.Now().UnixNano())

	var sb strings.Builder
	for i := 0; i < width; i++ {
		fmt.Fprintf(&sb, "%d", numeric[ rand.Intn(r) ])
	}
	return sb.String()
}

var r *rand.Rand

func RandString(len int) string {
	if r == nil {
		r = rand.New(rand.NewSource(time.Now().Unix()))
	}
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		b := r.Intn(26) + 65
		bytes[i] = byte(b)
	}
	return string(bytes)
}
