package gpu

//Thought it would be appropriate to put the sprite type code in a different file

import (
	"fmt"
	"github.com/dobyrch/termboy-go/types"
	"log"
)

type Sprite interface {
	UpdateSprite(addr types.Word, value byte)
	GetTileID(no int) int
	SpriteAttributes() *SpriteAttributes
	PushScanlines(fromScanline, amount int)
	PopScanline() (int, int)
	IsScanlineDrawQueueEmpty() bool
	ResetScanlineDrawQueue()
}

//8x8 Sprites!
type Sprite8x8 struct {
	SpriteAttrs       *SpriteAttributes
	TileID            int
	ScanlineDrawQueue []int
	CurrentTileLine   int
}

func NewSprite8x8() *Sprite8x8 {
	var sprite *Sprite8x8 = new(Sprite8x8)
	sprite.SpriteAttrs = new(SpriteAttributes)
	return sprite
}

func (s *Sprite8x8) String() string {
	return fmt.Sprintf("%s | Size: 8x8 | Tile ID: %v", s.SpriteAttrs, s.TileID)
}

func (s *Sprite8x8) SpriteAttributes() *SpriteAttributes {
	return s.SpriteAttrs
}

func (s *Sprite8x8) UpdateSprite(addr types.Word, value byte) {
	var spriteAttrId int = int(addr % 4)
	if spriteAttrId == 2 {
		s.TileID = int(value)
	} else {
		s.SpriteAttrs.Update(spriteAttrId, value)
	}
}

func (s *Sprite8x8) PushScanlines(fromScanline, amount int) {
	for i := 0; i < amount; i++ {
		if len(s.ScanlineDrawQueue) == 8 {
			break
		}
		s.ScanlineDrawQueue = append(s.ScanlineDrawQueue, fromScanline+i)
	}
}

func (s *Sprite8x8) IsScanlineDrawQueueEmpty() bool {
	return len(s.ScanlineDrawQueue) == 0
}

func (s *Sprite8x8) PopScanline() (int, int) {
	if len(s.ScanlineDrawQueue) == 0 {
		log.Panicln("Scanline queue is empty!")
	}

	value := s.ScanlineDrawQueue[0]
	oldCurrentTileLine := s.CurrentTileLine
	s.ScanlineDrawQueue = s.ScanlineDrawQueue[1:]

	if s.CurrentTileLine < 7 {
		s.CurrentTileLine++
	}
	return value, oldCurrentTileLine
}

func (s *Sprite8x8) ResetScanlineDrawQueue() {
	if len(s.ScanlineDrawQueue) > 0 {
		s.ScanlineDrawQueue = make([]int, 0)
	}
	s.CurrentTileLine = 0
}

func (s *Sprite8x8) GetTileID(no int) int {
	if no > 0 {
		log.Panicln("8x8 sprites only consist of one tile")
	}
	return s.TileID
}

// 8x16 SPRITES!
type Sprite8x16 struct {
	SpriteAttrs       *SpriteAttributes
	TileIDs           [2]int
	ScanlineDrawQueue []int
	CurrentTileLine   int
}

func NewSprite8x16() *Sprite8x16 {
	var sprite *Sprite8x16 = new(Sprite8x16)
	sprite.SpriteAttrs = new(SpriteAttributes)
	return sprite
}

func (s *Sprite8x16) String() string {
	return fmt.Sprintf("%s | Size: 8x16 | Tile IDs: %v", s.SpriteAttrs, s.TileIDs)
}

func (s *Sprite8x16) UpdateSprite(addr types.Word, value byte) {
	var spriteAttrId int = int(addr % 4)
	if spriteAttrId == 2 {
		s.TileIDs[0] = int(value)
		s.TileIDs[1] = int(value + 1)
	} else {
		s.SpriteAttrs.Update(spriteAttrId, value)
	}
}

func (s *Sprite8x16) GetTileID(no int) int {
	if no > 1 {
		log.Panicln("8x16 sprites only consist of two tiles")
	}
	return s.TileIDs[no]
}

func (s *Sprite8x16) SpriteAttributes() *SpriteAttributes {
	return s.SpriteAttrs
}

func (s *Sprite8x16) PushScanlines(fromScanline, amount int) {
	for i := 0; i < amount; i++ {
		if len(s.ScanlineDrawQueue) == 16 {
			break
		}
		s.ScanlineDrawQueue = append(s.ScanlineDrawQueue, fromScanline+i)
	}
}

func (s *Sprite8x16) IsScanlineDrawQueueEmpty() bool {
	return len(s.ScanlineDrawQueue) == 0
}

func (s *Sprite8x16) PopScanline() (int, int) {
	if len(s.ScanlineDrawQueue) == 0 {
		log.Panicln("Scanline queue is empty!")
	}

	value := s.ScanlineDrawQueue[0]
	oldCurrentTileLine := s.CurrentTileLine
	s.ScanlineDrawQueue = s.ScanlineDrawQueue[1:]
	if s.CurrentTileLine < 15 {
		s.CurrentTileLine++
	}
	return value, oldCurrentTileLine
}

func (s *Sprite8x16) ResetScanlineDrawQueue() {
	if len(s.ScanlineDrawQueue) > 0 {
		s.ScanlineDrawQueue = make([]int, 0)
	}
	s.CurrentTileLine = 0
}

//Sprite attributes
type SpriteAttributes struct {
	Y                      int
	X                      int
	SpriteHasPriority      bool
	ShouldFlipVertically   bool
	ShouldFlipHorizontally bool
	NonCGBPaletteSelected  int
	CGBBankNo              int
	CGBPaletteNo           int
}

func (sa *SpriteAttributes) Update(attributeId int, fromValue byte) {
	switch attributeId {
	case 0:
		sa.Y = int(fromValue)
	case 1:
		sa.X = int(fromValue)
	case 3:
		if (fromValue & 0x80) != 0x80 {
			sa.SpriteHasPriority = true
		} else {
			sa.SpriteHasPriority = false
		}

		if (fromValue & 0x40) == 0x40 {
			sa.ShouldFlipVertically = true
		} else {
			sa.ShouldFlipVertically = false
		}

		if (fromValue & 0x20) == 0x20 {
			sa.ShouldFlipHorizontally = true
		} else {
			sa.ShouldFlipHorizontally = false
		}

		if (fromValue & 0x10) == 0x10 {
			sa.NonCGBPaletteSelected = 1
		} else {
			sa.NonCGBPaletteSelected = 0
		}

		//CGB attributes
		sa.CGBBankNo = int(fromValue&0x08) >> 3
		sa.CGBPaletteNo = int(fromValue & 0x07)
	}
}

func (s *SpriteAttributes) String() string {
	return fmt.Sprintf("%#v", s)
}
