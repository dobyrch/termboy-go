package gpu

import (
	"github.com/dobyrch/termboy-go/components"
	"github.com/dobyrch/termboy-go/constants"
	"fmt"
	"log"
	"github.com/dobyrch/termboy-go/types"
)

const NAME = "GPU"
const PREFIX = NAME + ":"

const DISPLAY_WIDTH int = 160
const DISPLAY_HEIGHT int = 144

const (
	TILEMAP0  types.Word = 0x9800
	TILEMAP1             = 0x9C00
	TILEDATA0            = 0x8800
	TILEDATA1            = 0x8000
)

//Standard graphics register addresses
const (
	LCDC            types.Word = 0xFF40
	STAT                       = 0xFF41
	SCROLLY                    = 0xFF42
	SCROLLX                    = 0xFF43
	LY                         = 0xFF44
	LYC                        = 0xFF45
	BGP                        = 0xFF47
	OBJECTPALETTE_0            = 0xFF48
	OBJECTPALETTE_1            = 0xFF49
	WX                         = 0xFF4B
	WY                         = 0xFF4A
)

const HBLANK byte = 0x00
const VBLANK byte = 0x01
const OAMREAD byte = 0x02
const VRAMREAD byte = 0x03
const Sprite8x16Mode byte = 0
const Sprite8x8Mode byte = 1

var GBColours []types.RGB = []types.RGB{
	types.RGB{Red: 235, Green: 235, Blue: 235},
	types.RGB{Red: 196, Green: 196, Blue: 196},
	types.RGB{Red: 96, Green: 96, Blue: 96},
	types.RGB{Red: 0, Green: 0, Blue: 0},
}

type RawTile [16]byte
type Tile [8][8]int
type Palette [4]types.RGB

type GPU struct {
	screenData            types.Screen
	rawScreenDotData      [144][160]int
	screenOutputChannel   chan *types.Screen
	irqHandler            components.IRQHandler
	vram                  [2][8192]byte
	oamRam                [160]byte
	vBlankInterruptThrown bool
	lcdInterruptThrown    bool

	mode                         byte
	clock                        int
	ly                           int
	lcdc                         byte
	lyc                          byte
	stat                         byte
	scrollY                      byte
	scrollX                      byte
	windowX                      byte
	windowY                      byte
	bgp                          byte
	obp0                         byte
	obp1                         byte
	cgbVramBankSelectionRegister byte
	RunningColorGBHardware       bool
	currentTileLineDotData       *[8]int

	bgrdOn         bool
	spritesOn      bool
	windowOn       bool
	displayOn      bool
	tileDataSelect types.Word
	spriteSizeMode byte

	bgTilemap     types.Word
	windowTilemap types.Word
	rawTiledata   [2][512]RawTile
	tiledata      [2][512]Tile
	sprites8x8    [40]Sprite
	sprites8x16   [40]Sprite

	bgPalette      Palette
	objectPalettes [2]Palette

	cgbBackgroundPalettes             [8]CGBPalette
	cgbObjectPalettes                 [8]CGBPalette
	cgbBGPWriteSpecReg                CGBPaletteSpecRegister
	cgbBGPWriteDataRegister           byte
	cgbOBJPWriteSpecReg               CGBPaletteSpecRegister
	cgbOBJPWriteDataRegister          byte
	cgbScreenPixelBackgroundTileAttrs [144][160]*CGBBackgroundTileAttrs
}

func NewGPU() *GPU {
	var g *GPU = new(GPU)
	g.Reset()
	return g
}

func (g *GPU) LinkScreen(screenChannel chan *types.Screen) {
	g.screenOutputChannel = screenChannel
	log.Println(PREFIX, "Linked screen to GPU")
}

func (g *GPU) LinkIRQHandler(m components.IRQHandler) {
	g.irqHandler = m
	log.Println(PREFIX, "Linked IRQ Handler to GPU")
}

func (g *GPU) Name() string {
	return NAME
}

func (g *GPU) Reset() {
	log.Println(PREFIX, "Resetting", g.Name())
	g.Write(LCDC, 0x00)
	g.screenData = *new(types.Screen)
	g.rawScreenDotData = *new([144][160]int)
	g.mode = 0
	g.ly = 0
	g.clock = 0
	g.vBlankInterruptThrown = false
	g.lcdInterruptThrown = false
	g.RunningColorGBHardware = false

	for i := 0; i < 40; i++ {
		g.sprites8x8[i] = NewSprite8x8()
		g.sprites8x16[i] = NewSprite8x16()
	}

	g.cgbBGPWriteSpecReg = *new(CGBPaletteSpecRegister)
	g.cgbOBJPWriteSpecReg = *new(CGBPaletteSpecRegister)
	g.cgbBackgroundPalettes = *new([8]CGBPalette)
	g.cgbObjectPalettes = *new([8]CGBPalette)
	g.currentTileLineDotData = new([8]int)
}

func (g *GPU) Step(t int) {
	if !g.displayOn {
		g.ly = 0
		g.clock = 456
		g.mode = HBLANK
	} else {
		if g.ly >= 144 {
			g.mode = VBLANK
			g.lcdInterruptThrown = false
		} else if g.clock >= 456-80 {
			g.mode = OAMREAD
			g.lcdInterruptThrown = false
		} else if g.clock >= 456-80-172 {
			g.mode = VRAMREAD
			g.lcdInterruptThrown = false
		} else {
			g.mode = HBLANK
			//throw HBlank LCD interrupt (if enabled)
			if g.HblankLCDInterruptEnabled() && g.lcdInterruptThrown == false {
				g.irqHandler.RequestInterrupt(constants.LCD_IRQ)
				g.lcdInterruptThrown = true
			}
		}
	}

	if !g.displayOn {
		return
	}

	g.clock -= t

	if g.clock <= 0 {
		g.clock += 456
		g.ly += 1

		if g.ly == 144 {
			//reset sprite draw queues after frame has been rendered
			for _, s := range g.sprites8x8 {
				s.ResetScanlineDrawQueue()
			}

			for _, s := range g.sprites8x16 {
				s.ResetScanlineDrawQueue()
			}

			//throw vblank interrupt
			if g.vBlankInterruptThrown == false {
				g.irqHandler.RequestInterrupt(constants.V_BLANK_IRQ)

				//throw VBLANK LCD interrupt (if enabled)
				if g.VBlankLCDInterruptEnabled() {
					g.irqHandler.RequestInterrupt(constants.LCD_IRQ)
				}
				g.vBlankInterruptThrown = true
			}

			//dump output to screen controller over a channel
			g.screenOutputChannel <- &g.screenData
		} else if g.ly > 153 {
			g.vBlankInterruptThrown = false
			g.ly = 0
		}

		//throw coincidence LCD interrupt (if enabled)
		if g.CoincidenceLCDInterruptEnabled() && byte(g.ly) == g.lyc {
			g.stat |= 0x04
			g.irqHandler.RequestInterrupt(constants.LCD_IRQ)
		}

		//Render scanline
		if g.ly < 144 {
			if g.displayOn {
				if g.bgrdOn {
					g.RenderBackgroundScanline()
				}

				if g.windowOn {
					g.RenderWindowScanline()
				}

				if g.spritesOn {
					g.RenderSpritesOnScanline()
				}
			}
		}
	}
}

func (g *GPU) CoincidenceLCDInterruptEnabled() bool {
	return (g.Read(STAT) & 0x40) == 0x40
}

func (g *GPU) VBlankLCDInterruptEnabled() bool {
	return (g.Read(STAT) & 0x10) == 0x10
}

func (g *GPU) HblankLCDInterruptEnabled() bool {
	return (g.Read(STAT) & 0x08) == 0x08
}

//Called from mmu
func (g *GPU) Write(addr types.Word, value byte) {
	switch {
	case addr >= 0x8000 && addr <= 0x9FFF:
		g.WriteToVideoRAM(addr, value)
	case addr >= 0xFE00 && addr <= 0xFE9F:
		g.oamRam[addr&0x009F] = value
		g.UpdateSprite(addr, value)
	default:
		switch addr {
		case LCDC:
			g.lcdc = value

			g.displayOn = value&0x80 == 0x80 //bit 7

			if value&0x40 == 0x40 { //bit 6
				g.windowTilemap = TILEMAP1
			} else {
				g.windowTilemap = TILEMAP0
			}

			g.windowOn = value&0x20 == 0x20 //bit 5

			if value&0x10 == 0x10 { //bit 4
				g.tileDataSelect = TILEDATA1
			} else {
				g.tileDataSelect = TILEDATA0
			}

			if value&0x08 == 0x08 { //bit 3
				g.bgTilemap = TILEMAP1
			} else {
				g.bgTilemap = TILEMAP0
			}

			if value&0x04 == 0x04 { //bit 2
				g.spriteSizeMode = Sprite8x16Mode
			} else {
				g.spriteSizeMode = Sprite8x8Mode
			}

			g.spritesOn = value&0x02 == 0x02 //bit 1
			g.bgrdOn = value&0x01 == 0x01    //bit 0
		case STAT:
			g.stat = (g.stat & 0x0F) | value
		case SCROLLY:
			g.scrollY = value
		case SCROLLX:
			g.scrollX = value
		case WX:
			g.windowX = value
		case WY:
			g.windowY = value
		case LY:
			g.ly = 0
		case LYC:
			g.lyc = value
		case BGP:
			g.bgp = value
			g.bgPalette = g.byteToPalette(value)
		case OBJECTPALETTE_0:
			g.obp0 = value
			g.objectPalettes[0] = g.byteToPalette(value)
		case OBJECTPALETTE_1:
			g.obp1 = value
			g.objectPalettes[1] = g.byteToPalette(value)
		case CGB_BGP_WRITESPEC_REGISTER:
			g.cgbBGPWriteSpecReg.Update(value)
		case CGB_BGP_WRITEDATA_REGISTER:
			g.cgbBGPWriteDataRegister = value
			if g.cgbBGPWriteSpecReg.High {
				g.cgbBackgroundPalettes[g.cgbBGPWriteSpecReg.PalleteNo].UpdateHigh(g.cgbBGPWriteSpecReg.PalleteDataNo, value)
			} else {
				g.cgbBackgroundPalettes[g.cgbBGPWriteSpecReg.PalleteNo].UpdateLow(g.cgbBGPWriteSpecReg.PalleteDataNo, value)
			}

			if g.cgbBGPWriteSpecReg.IncrementOnNext {
				g.cgbBGPWriteSpecReg.Increment()
			}
		case CGB_OBJP_WRITESPEC_REGISTER:
			g.cgbOBJPWriteSpecReg.Update(value)
		case CGB_OBJP_WRITEDATA_REGISTER:
			g.cgbOBJPWriteDataRegister = value
			if g.cgbOBJPWriteSpecReg.High {
				g.cgbObjectPalettes[g.cgbOBJPWriteSpecReg.PalleteNo].UpdateHigh(g.cgbOBJPWriteSpecReg.PalleteDataNo, value)
			} else {
				g.cgbObjectPalettes[g.cgbOBJPWriteSpecReg.PalleteNo].UpdateLow(g.cgbOBJPWriteSpecReg.PalleteDataNo, value)
			}

			if g.cgbOBJPWriteSpecReg.IncrementOnNext {
				g.cgbOBJPWriteSpecReg.Increment()
			}
		case CGB_VRAM_BANK_SELECT:
			g.cgbVramBankSelectionRegister = value
		default:
			log.Printf(PREFIX+" WARNING: cannot write to register address %s as it is unknown", addr)
		}
	}
}

//Called from mmu
func (g *GPU) Read(addr types.Word) byte {
	switch {
	case addr >= 0x8000 && addr <= 0x9FFF:
		return g.ReadFromVideoRAM(addr)
	case addr >= 0xFE00 && addr <= 0xFE9F:
		return g.oamRam[addr&0x009F]
	default:
		switch addr {
		case LCDC:
			return g.lcdc
		case STAT:
			return (g.mode | g.stat&0xF8)
		case SCROLLY:
			return g.scrollY
		case SCROLLX:
			return g.scrollX
		case LY:
			return byte(g.ly)
		case LYC:
			return g.lyc
		case BGP:
			return g.bgp
		case OBJECTPALETTE_0:
			return g.obp0
		case OBJECTPALETTE_1:
			return g.obp1
		case WX:
			return g.windowX
		case WY:
			return g.windowY
		case CGB_BGP_WRITESPEC_REGISTER:
			return g.cgbBGPWriteSpecReg.Value
		case CGB_BGP_WRITEDATA_REGISTER:
			//When the write data register is read, the data at the address specified by the write-specification register is returned
			//so we have to convert the color back to high/low values and return the values
			var paletteColor CGBColor = g.cgbBackgroundPalettes[g.cgbBGPWriteSpecReg.PalleteNo][g.cgbBGPWriteSpecReg.PalleteDataNo]
			if g.cgbBGPWriteSpecReg.High {
				return paletteColor.High()
			} else {
				return paletteColor.Low()
			}
		case CGB_OBJP_WRITESPEC_REGISTER:
			return g.cgbOBJPWriteSpecReg.Value
		case CGB_OBJP_WRITEDATA_REGISTER:
			//When the write data register is read, the data at the address specified by the write-specification register is returned
			//so we have to convert the color back to high/low values and return the values
			var paletteColor CGBColor = g.cgbObjectPalettes[g.cgbOBJPWriteSpecReg.PalleteNo][g.cgbOBJPWriteSpecReg.PalleteDataNo]
			if g.cgbOBJPWriteSpecReg.High {
				return paletteColor.High()
			} else {
				return paletteColor.Low()
			}
		case CGB_VRAM_BANK_SELECT:
			return g.cgbVramBankSelectionRegister
		default:
			log.Printf(PREFIX+" WARNING: register address %s unknown", addr)
		}
	}

	return 0x00
}

func (g *GPU) WriteToVideoRAM(addr types.Word, value byte) {
	var bankAddr types.Word = addr & 0x1FFF
	if g.RunningColorGBHardware {
		//CGB has two banks of 8KB VRAM
		bankSelection := g.cgbVramBankSelectionRegister & 0x01
		g.vram[bankSelection][bankAddr] = value
		g.UpdateTile(addr, value, bankSelection)
	} else {
		g.vram[0][bankAddr] = value
		g.UpdateTile(addr, value, 0)
	}
}

func (g *GPU) ReadFromVideoRAM(addr types.Word) byte {
	var bankAddr types.Word = addr & 0x1FFF
	if g.RunningColorGBHardware {
		//CGB has two banks of 8KB VRAM
		bankSelection := g.cgbVramBankSelectionRegister & 0x01
		return g.vram[bankSelection][bankAddr]
	} else {
		return g.vram[0][bankAddr]
	}
}

func (g *GPU) UpdateSprite(addr types.Word, value byte) {
	var spriteId types.Word = (addr & 0x00FF) / 4
	if g.spriteSizeMode == Sprite8x8Mode {
		g.sprites8x8[spriteId].UpdateSprite(addr, value)
	} else {
		g.sprites8x16[spriteId].UpdateSprite(addr, value)
	}
}

//Update the tile at address with value
func (g *GPU) UpdateTile(addr types.Word, value byte, bank byte) {
	//get the ID of the tile being updated (between 0 and 383)
	var tileId types.Word = ((addr & 0x1FFF) >> 4) & 511
	g.rawTiledata[bank][tileId][addr%16] = value

	recalcTile := func(rawtile RawTile) Tile {
		var tile Tile
		for tileY := 0; tileY < 8; tileY++ {
			lineLo, lineHi := int(rawtile[tileY*2]), int(rawtile[(tileY*2)+1])
			var tileX uint = 0
			for ; tileX < 8; tileX++ {
				tile[tileY][tileX] = ((lineLo >> (7 - tileX) & 1) | (lineHi>>(7-tileX)&1)<<1)
			}
		}

		return tile
	}

	g.tiledata[bank][tileId] = recalcTile(g.rawTiledata[bank][tileId])
}

func (g *GPU) RenderBackgroundScanline() {
	//find where in the tile map we are related to the current scan line + scroll Y (wraps around)
	screenYAdjusted := int(g.ly) + int(g.scrollY)
	var initialTilemapOffset types.Word = g.bgTilemap + types.Word(screenYAdjusted)%256/8*32
	var initialLineOffset types.Word = types.Word(g.scrollX) / 8 % 32

	//find where in the tile we are
	initialTileX := int(g.scrollX) % 8
	initialTileY := screenYAdjusted % 8

	//screen will always draw from X = 0
	g.DrawScanline(initialTilemapOffset, initialLineOffset, 0, initialTileX, initialTileY)
}

func (g *GPU) RenderWindowScanline() {
	screenYAdjusted := g.ly - int(g.windowY)

	if (g.windowX >= 0 && g.windowX < 167) && (g.windowY >= 0 && g.windowY < 144) && screenYAdjusted >= 0 {
		var initialTilemapOffset types.Word = g.windowTilemap + types.Word(screenYAdjusted)/8*32
		var initialLineOffset types.Word = 0
		var screenXAdjusted int = int((g.windowX - 7) % 255)

		//find where in the tile we are
		initialTileX := screenXAdjusted % 8
		initialTileY := screenYAdjusted % 8

		g.DrawScanline(initialTilemapOffset, initialLineOffset, screenXAdjusted, initialTileX, initialTileY)
	}
}

func (g *GPU) DrawScanline(tilemapOffset, lineOffset types.Word, screenX, tileX, tileY int) {
	if g.RunningColorGBHardware {
		g.drawCGBScanline(tilemapOffset, lineOffset, screenX, tileX, tileY)
	} else {
		g.drawNonCGBScanline(tilemapOffset, lineOffset, screenX, tileX, tileY)
	}
}

func (g *GPU) drawCGBScanline(tilemapOffset, lineOffset types.Word, screenX, tileX, tileY int) {
	//get tile attributes to start from
	tileNumber, tileInfo := g.getCGBBackgroundTileAttrs(tilemapOffset, lineOffset)

	for ; screenX < DISPLAY_WIDTH; screenX++ {
		var t *Tile = &g.tiledata[tileInfo.BankNo][tileNumber]
		formatTileLine(t, tileY, tileInfo.FlipHorizontally, tileInfo.FlipVertically, g.currentTileLineDotData)

		//draw the pixel to the screenData data buffer (running through the color palette)
		g.screenData[g.ly][screenX] = g.cgbBackgroundPalettes[tileInfo.PaletteNo][g.currentTileLineDotData[tileX]].ToRGB()
		g.rawScreenDotData[g.ly][screenX] = g.currentTileLineDotData[tileX]
		g.cgbScreenPixelBackgroundTileAttrs[g.ly][screenX] = tileInfo

		//move along line in tile until you reach the end
		tileX++
		if tileX == 8 {
			tileX = 0
			lineOffset = (lineOffset + 1) % 32

			//get next tile in line
			tileNumber, tileInfo = g.getCGBBackgroundTileAttrs(tilemapOffset, lineOffset)
		}
	}
}

func (g *GPU) drawNonCGBScanline(tilemapOffset, lineOffset types.Word, screenX, tileX, tileY int) {
	//get tile to start from
	tileId := g.calculateTileNo(tilemapOffset, lineOffset)

	for ; screenX < DISPLAY_WIDTH; screenX++ {
		//draw the pixel to the screenData data buffer (running through the bgPalette)
		color := g.bgPalette[g.tiledata[0][tileId][tileY][tileX]]
		g.screenData[g.ly][screenX] = color
		g.rawScreenDotData[g.ly][screenX] = g.tiledata[0][tileId][tileY][tileX]

		//move along line in tile until you reach the end
		tileX++
		if tileX == 8 {
			tileX = 0
			lineOffset = (lineOffset + 1) % 32

			//get next tile in line
			tileId = g.calculateTileNo(tilemapOffset, lineOffset)
		}
	}
}

//method to calculate the tilenumber within the tilemap
func (g *GPU) calculateTileNo(tilemapOffset types.Word, lineOffset types.Word) int {
	tileId := int(g.Read(types.Word(tilemapOffset + lineOffset)))

	//if tile data is 0 then it is signed
	if g.tileDataSelect == TILEDATA0 {
		if tileId < 128 {
			tileId += 256
		}
	}
	return tileId
}

//CGB has additional attributes in bank 1 for each background tile
func (g *GPU) getCGBBackgroundTileAttrs(tilemapOffset types.Word, lineOffset types.Word) (int, *CGBBackgroundTileAttrs) {
	if g.RunningColorGBHardware {
		var currentSelectedBankTmp byte = g.cgbVramBankSelectionRegister

		//tile number data always comes from bank 0
		g.cgbVramBankSelectionRegister = 0
		var tileNo int = g.calculateTileNo(tilemapOffset, lineOffset)

		//tile attribute data always comes from bank 1
		g.cgbVramBankSelectionRegister = 1
		var attributeData byte = g.Read(types.Word(tilemapOffset + lineOffset))

		//revert bank selection register to what it was set to previously
		g.cgbVramBankSelectionRegister = currentSelectedBankTmp

		return tileNo, NewCGBBackgroundTileAttrs(attributeData)
	} else {
		panic("Cannot call this function, not in color gb mode!")
	}

	return -1, nil
}

func (g *GPU) RenderSpritesOnScanline() {
	if g.spriteSizeMode == Sprite8x8Mode {
		for _, sprite := range g.sprites8x8 {
			if sprite.SpriteAttributes().X != 0x00 && sprite.SpriteAttributes().Y != 0x00 {
				//if sprite is on current scanline, then it needs to be drawn over the next
				//8 lines, so these are added to a queue
				if sprite.SpriteAttributes().Y-16 <= 0 {
					sprite.PushScanlines(g.ly, 8)
				} else if sprite.SpriteAttributes().Y-16 == g.ly {
					sprite.PushScanlines(g.ly, 8)
				}

				if sprite.IsScanlineDrawQueueEmpty() == false {
					//if next scanline == LY, then draw the line
					if scanline, tileLine := sprite.PopScanline(); scanline == g.ly {
						g.DrawSpriteTileLine(sprite, sprite.GetTileID(0), 0, tileLine)
					}
				}
			}
		}
	} else {
		for _, sprite := range g.sprites8x16 {
			if sprite.SpriteAttributes().X != 0x00 && sprite.SpriteAttributes().Y != 0x00 {
				//if sprite is on current scanline, then it needs to be drawn over the next
				//16 lines, so these are added to a queue
				if sprite.SpriteAttributes().Y-16 <= 0 {
					sprite.PushScanlines(g.ly, 16)
				} else if sprite.SpriteAttributes().Y-16 == g.ly {
					sprite.PushScanlines(g.ly, 16)
				}

				if sprite.IsScanlineDrawQueueEmpty() == false {
					//if next scanline == LY, then draw the line
					if scanline, tileLine := sprite.PopScanline(); scanline == g.ly {
						if sprite.SpriteAttributes().ShouldFlipVertically {
							if tileLine < 8 {
								g.DrawSpriteTileLine(sprite, sprite.GetTileID(1), 0, tileLine) //draw second portion of sprite using next tile 8 lines down
							} else {
								g.DrawSpriteTileLine(sprite, sprite.GetTileID(0), 8, tileLine-8)
							}
						} else {
							if tileLine < 8 {
								g.DrawSpriteTileLine(sprite, sprite.GetTileID(0), 0, tileLine)
							} else {
								g.DrawSpriteTileLine(sprite, sprite.GetTileID(1), 8, tileLine-8) //draw second portion of sprite using next tile 8 lines down
							}
						}
					}
				}
			}
		}
	}
}

//TODO: Sprite precedence rules
// Draws a tile for the given sprite. Only draws one tile
func (g *GPU) DrawSpriteTileLine(s Sprite, tileId, screenYOffset, tileY int) {
	if g.RunningColorGBHardware {
		g.drawCGBSpriteTileLine(s, tileId, screenYOffset, tileY)
	} else {
		g.drawNonCGBSpriteTileLine(s, tileId, screenYOffset, tileY)
	}
}

func (g *GPU) drawCGBSpriteTileLine(s Sprite, tileId, screenYOffset, tileY int) {
	if s.SpriteAttributes().X >= 0 && s.SpriteAttributes().Y >= 0 {
		//tile data can come from one of two banks in CGB mode
		var t *Tile = &g.tiledata[s.SpriteAttributes().CGBBankNo][tileId]
		formatTileLine(t, tileY, s.SpriteAttributes().ShouldFlipHorizontally, s.SpriteAttributes().ShouldFlipVertically, g.currentTileLineDotData)

		sx, sy := s.SpriteAttributes().X-8, s.SpriteAttributes().Y-16
		for tileX := 0; tileX < 8; tileX++ {
			if g.currentTileLineDotData[tileX] != 0 {
				adjX, adjY := sx+tileX, sy+tileY+screenYOffset
				if (adjY < DISPLAY_HEIGHT && adjY >= 0) && (adjX < DISPLAY_WIDTH && adjX >= 0) {
					//if background tile has priority then skip drawing this sprite pixel

					if g.bgrdOn {
						bgDotData := g.rawScreenDotData[adjY][adjX]
						objDotData := g.currentTileLineDotData[tileX]
						result := calculateObjToBackgroundPriority(g.cgbScreenPixelBackgroundTileAttrs[adjY][adjX].HasPriority, s.SpriteAttributes().SpriteHasPriority, bgDotData, objDotData)
						if result != OBJ_PRIORITY {
							continue
						}
					}

					g.screenData[adjY][adjX] = g.cgbObjectPalettes[s.SpriteAttributes().CGBPaletteNo][g.currentTileLineDotData[tileX]].ToRGB()
				}
			}
		}
	}
}

func (g *GPU) drawNonCGBSpriteTileLine(s Sprite, tileId, screenYOffset, tileY int) {
	if s.SpriteAttributes().X >= 0 && s.SpriteAttributes().Y >= 0 {
		//tile data in non CGB mode only comes from bank 0
		var t *Tile = &g.tiledata[0][tileId]
		formatTileLine(t, tileY, s.SpriteAttributes().ShouldFlipHorizontally, s.SpriteAttributes().ShouldFlipVertically, g.currentTileLineDotData)

		sx, sy := s.SpriteAttributes().X-8, s.SpriteAttributes().Y-16
		for tileX := 0; tileX < 8; tileX++ {
			if g.currentTileLineDotData[tileX] != 0 {
				adjX, adjY := sx+tileX, sy+tileY+screenYOffset
				if (adjY < DISPLAY_HEIGHT && adjY >= 0) && (adjX < DISPLAY_WIDTH && adjX >= 0) {
					//If sprite does NOT have priority and background palette color at coordinate (adjX, adjY) isn't in bgp[0], then skip drawing pixel
					if !s.SpriteAttributes().SpriteHasPriority && g.screenData[adjY][adjX] != g.bgPalette[0] {
						continue
					}

					g.screenData[adjY][adjX] = g.objectPalettes[s.SpriteAttributes().NonCGBPaletteSelected][g.currentTileLineDotData[tileX]]
				}
			}
		}
	}
}

func (g *GPU) byteToPalette(b byte) Palette {
	var palette Palette
	palette[0] = GBColours[int(b&0x03)]
	palette[1] = GBColours[int((b>>2)&0x03)]
	palette[2] = GBColours[int((b>>4)&0x03)]
	palette[3] = GBColours[(int(b>>6) & 0x03)]
	return palette
}

//debug helpers
func (g *GPU) DumpTiles() [512][8][8]types.RGB {
	fmt.Println("Dumping", len(g.tiledata[0]), "tiles")
	var out [512][8][8]types.RGB
	for i, tile := range g.tiledata[0] {
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				cr := GBColours[tile[y][x]]
				out[i][y][x] = cr
			}
		}
	}

	return out
}

func (g *GPU) Dump8x8Sprites() [40][8][8]types.RGB {
	fmt.Println("Dumping", len(g.sprites8x8), "sprites")
	var out [40][8][8]types.RGB
	for i, spr := range g.sprites8x8 {
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				tileId := spr.GetTileID(0)
				tile := g.tiledata[0][tileId]
				cr := GBColours[tile[y][x]]
				out[i][y][x] = cr
			}
		}
	}
	return out
}

func (g *GPU) DumpTilemap(tileMapAddr types.Word, tileDataSigned bool) [256][256]types.RGB {
	fmt.Print("Dumping Tilemap ", tileMapAddr)
	if tileDataSigned {
		fmt.Println(" (signed)")
	} else {
		fmt.Println(" (unsigned)")
	}

	var result [256][256]types.RGB
	var tileMapAddrOffset types.Word = tileMapAddr
	var rx int = 0
	var ry int = 0

	for lineX := 0; lineX < 32; lineX++ {
		for tileY := 0; tileY < 8; tileY++ {
			for lineY := 0; lineY < 32; lineY++ {
				tileId := int(g.Read(tileMapAddrOffset + types.Word(lineY)))
				if tileDataSigned {
					if tileId < 128 {
						tileId += 256
					}
				}
				tile := g.tiledata[0][tileId]
				for tileX := 0; tileX < 8; tileX++ {
					cr := GBColours[tile[tileY][tileX]]
					result[rx][ry] = cr
					rx++
				}
			}
			rx = 0
			ry++
		}
		tileMapAddrOffset += types.Word(32)
	}
	return result
}

// (Static method) Format a tile according to the vertical/horizontal flipping instructions
func formatTileLine(t *Tile, tileY int, flipHorizontal, flipVertical bool, tileLineStorageArea *[8]int) {
	//flip both
	if flipVertical && flipHorizontal {
		for x := 0; x < 8; x++ {
			tileLineStorageArea[x] = t[7-tileY][7-x]
		}
		return
	}

	if flipVertical {
		for i, v := range t[7-tileY] {
			tileLineStorageArea[i] = v
		}
		return
	}

	if flipHorizontal {
		for x := 0; x < 8; x++ {
			tileLineStorageArea[x] = t[tileY][7-x]
		}
		return
	}

	for i, v := range t[tileY] {
		tileLineStorageArea[i] = v
	}
}
