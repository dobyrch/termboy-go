package utils

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ByteToString(b byte) string {
	var zeroes string
	if b < 0x10 {
		zeroes += "0"
	}
	return fmt.Sprintf("0x%s%X", zeroes, b)
}

func StringToByte(s string) (byte, error) {
	s = strings.Replace(s, "0x", "", 1)
	if len(s) > 2 {
		return 0x0, errors.New("Please enter an address between 00 and FF")
	}

	result, err := strconv.ParseInt(s, 16, 64)

	return byte(result), err
}

func StringToWord(s string) (uint16, error) {
	s = strings.Replace(s, "0x", "", 1)

	if len(s) > 4 {
		return 0x0, errors.New("Please enter an address between 0000 and FFFF")
	}

	result, err := strconv.ParseInt(s, 16, 64)

	return uint16(result), err
}

//Joins two bytes together to form a 16 bit integer
func JoinBytes(hob, lob byte) uint16 {
	return (uint16(hob) << 8) ^ uint16(lob)
}

//Splits one 16 bit integer to two bytes
func SplitIntoBytes(bb uint16) (byte, byte) {
	return byte(bb >> 8), byte(bb & 0x00FF)
}

//swaps the nibbles of a byte around
func SwapNibbles(a byte) byte {
	return (a&0xF0)>>4 ^ ((a & 0x0F) << 4)
}

func CompareBytes(a, b byte, operator string) bool {
	switch operator {
	case "==":
		return a == b
	case ">":
		return a > b
	case "<":
		return a < b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	}
	return false
}

func CompareWords(a, b uint16, operator string) bool {
	switch operator {
	case "==":
		return a == b
	case ">":
		return a > b
	case "<":
		return a < b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	}
	return false
}

func PadRight(s string, maxLen int, padStr string) string {
	if l := len(s); l < maxLen {
		padAmount := maxLen - l
		s = s + strings.Repeat(padStr, padAmount)
	}
	return s
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
