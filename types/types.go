package types

import "fmt"

type RGB struct {
	Red   byte
	Green byte
	Blue  byte
}

type Screen [144][160]RGB

type Register byte
type Word uint16
type Words []Word

func (w Word) String() string {
	var zeroes string
	switch {
	case w < 0x0010:
		zeroes += "000"
	case w < 0x0100:
		zeroes += "00"
	case w < 0x1000:
		zeroes += "0"
	}
	return fmt.Sprintf("0x%s%X", zeroes, uint16(w))
}

func (w Words) Len() int {
	return len(w)
}

func (w Words) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}

func (w Words) Less(i, j int) bool {
	return w[i] < w[j]
}
