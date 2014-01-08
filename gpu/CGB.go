package gpu

import (
	"fmt"
	"github.com/dobyrch/termboy-go/types"
)

const OBJ_PRIORITY = 1
const BG_PRIORITY = 2

//Colour GB graphics register addresses
const (
	CGB_VRAM_BANK_SELECT        types.Word = 0xFF4F
	CGB_BGP_WRITESPEC_REGISTER             = 0xFF68
	CGB_BGP_WRITEDATA_REGISTER             = 0xFF69
	CGB_OBJP_WRITESPEC_REGISTER            = 0xFF6A
	CGB_OBJP_WRITEDATA_REGISTER            = 0xFF6B
)

//Represents the attribute data for a background tile
type CGBBackgroundTileAttrs struct {
	HasPriority      bool
	FlipHorizontally bool
	FlipVertically   bool
	BankNo           int
	PaletteNo        int
}

func NewCGBBackgroundTileAttrs(attributeData byte) *CGBBackgroundTileAttrs {
	var cbc *CGBBackgroundTileAttrs = new(CGBBackgroundTileAttrs)
	cbc.HasPriority = (attributeData & 0x80) == 0x80
	cbc.FlipVertically = (attributeData & 0x40) == 0x40
	cbc.FlipHorizontally = (attributeData & 0x20) == 0x20
	cbc.BankNo = int((attributeData & 0x08) >> 3)
	cbc.PaletteNo = int(attributeData & 0x07)
	return cbc
}

func (cattr *CGBBackgroundTileAttrs) String() string {
	return fmt.Sprintf("%#v", cattr)
}

//Represents a color
type CGBColor types.Word

func (c CGBColor) ToRGB() types.RGB {
	return types.RGB{
		Red:   byte(c&0x001F) * 8,
		Green: byte(c&0x03E0>>5) * 8,
		Blue:  byte(c&0x7C00>>10) * 8}
}

func (c CGBColor) High() byte {
	return byte((c & 0xFF00) >> 8)
}

func (c CGBColor) Low() byte {
	return byte(c & 0x00FF)
}

//Represents a color palette (4 colors)
type CGBPalette [4]CGBColor

func (cp *CGBPalette) UpdateHigh(colorNo int, value byte) {
	cp[colorNo] = (cp[colorNo] & 0x00FF) | CGBColor(value)<<8
}

func (cp *CGBPalette) UpdateLow(colorNo int, value byte) {
	cp[colorNo] = (cp[colorNo] & 0xFF00) | CGBColor(value)
}

//Represents the write specification register for a color palette
type CGBPaletteSpecRegister struct {
	Value           byte
	PalleteNo       int
	PalleteDataNo   int
	High            bool
	IncrementOnNext bool
}

func (psr *CGBPaletteSpecRegister) Update(value byte) {
	psr.Value = value
	psr.PalleteNo = int((value & 0x38) >> 3)
	psr.PalleteDataNo = int((value & 0x06) >> 1)
	psr.High = (value & 0x01) == 0x01
	psr.IncrementOnNext = (value & 0x80) == 0x80
}

func (psr *CGBPaletteSpecRegister) Increment() {
	psr.Update(psr.Value + 1)
}

/*
Calculate the priority of BG vs OBJ using rules defined in
the Gameboy Programming Manual - Page 67
*/
func calculateObjToBackgroundPriority(backgroundPriorityFlag, objPriorityFlag bool, bgDotData, objDotData int) int {
	if backgroundPriorityFlag { //BG has highest priority
		switch {
		case objDotData == 0x00 && bgDotData == 0x00: //bg gets priority
			return BG_PRIORITY
		case objDotData == 0x00 && bgDotData != 0x00: //bg gets priority
			return BG_PRIORITY
		case objDotData != 0x00 && bgDotData == 0x00: //object gets priority
			return OBJ_PRIORITY
		case objDotData != 0x00 && bgDotData != 0x00: //bg gets priority
			return BG_PRIORITY
		}
	} else { // bg depends on what obj priority is
		if objPriorityFlag { //OBJ has highest priority
			switch {
			case objDotData == 0x00 && bgDotData == 0x00: //bg gets priority
				return BG_PRIORITY
			case objDotData == 0x00 && bgDotData != 0x00: //bg gets priority
				return BG_PRIORITY
			case objDotData != 0x00 && bgDotData == 0x00: //object gets priority
				return OBJ_PRIORITY
			case objDotData != 0x00 && bgDotData != 0x00: //object gets priority
				return OBJ_PRIORITY
			}
		} else {
			switch { //give priority to BG
			case objDotData == 0x00 && bgDotData == 0x00: //bg gets priority
				return BG_PRIORITY
			case objDotData == 0x00 && bgDotData != 0x00: //bg gets priority
				return BG_PRIORITY
			case objDotData != 0x00 && bgDotData == 0x00: //object gets priority
				return OBJ_PRIORITY
			case objDotData != 0x00 && bgDotData != 0x00: //bg gets priority
				return BG_PRIORITY
			}
		}

	}

	return OBJ_PRIORITY
}
