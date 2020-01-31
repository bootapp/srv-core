package utils

import (
	"strconv"
	"strings"
)

func StringIpToInt(ipStr string) int32 {
	segments := strings.Split(ipStr, ".")
	if len(segments) != 4 {
		return 0
	}
	var ipInt = 0
	var pos uint = 24
	for _, seg := range segments {
		tempInt, _ := strconv.Atoi(seg)
		tempInt = tempInt << pos
		ipInt = ipInt | tempInt
		pos -= 8
	}
	return int32(ipInt)
}
