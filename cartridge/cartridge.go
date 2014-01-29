package cartridge

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/dobyrch/termboy-go/utils"
	"os"
	"path/filepath"
	"strings"
)

const (
	MBC_0                 = 0x00
	MBC_1                 = 0x01
	MBC_1_RAM             = 0x02
	MBC_1_RAM_BATT        = 0x03
	MBC_3_RAM_BATT_TIMER  = 0x10
	MBC_3_RAM_BATT        = 0x13
	MBC_5                 = 0x19
	MBC_5_RAM             = 0x1A
	MBC_5_RAM_BATT        = 0x1B
	MBC_5_RUMBLE          = 0x1C
	MBC_5_RAM_RUMBLE      = 0x1D
	MBC_5_RAM_BATT_RUMBLE = 0x1E
)

type CartridgeType struct {
	ID          byte
	Description string
}

var CartridgeTypes map[byte]CartridgeType = map[byte]CartridgeType{
	MBC_0:                 CartridgeType{MBC_0, "ROM ONLY"},
	MBC_1:                 CartridgeType{MBC_1, "ROM+MBC1"},
	MBC_1_RAM:             CartridgeType{MBC_1_RAM, "ROM+MBC1+RAM"},
	MBC_1_RAM_BATT:        CartridgeType{MBC_1_RAM_BATT, "ROM+MBC1+RAM+BATT"},
	MBC_3_RAM_BATT_TIMER:  CartridgeType{MBC_3_RAM_BATT_TIMER, "ROM+MBC3+RAM+BATT+TIMER"},
	MBC_3_RAM_BATT:        CartridgeType{MBC_3_RAM_BATT, "ROM+MBC3+RAM+BATT"},
	MBC_5:                 CartridgeType{MBC_5, "ROM+MBC5"},
	MBC_5_RAM:             CartridgeType{MBC_5_RAM, "ROM+MBC5+RAM"},
	MBC_5_RAM_BATT:        CartridgeType{MBC_5_RAM_BATT, "ROM+MBC5+RAM+BATT"},
	MBC_5_RUMBLE:          CartridgeType{MBC_5_RUMBLE, "ROM+MBC5+RUMBLE"},
	MBC_5_RAM_RUMBLE:      CartridgeType{MBC_5_RAM_RUMBLE, "ROM+MBC5+RAM+RUMBLE"},
	MBC_5_RAM_BATT_RUMBLE: CartridgeType{MBC_5_RAM_BATT_RUMBLE, "ROM+MBC5+RAM+BATT+RUMBLE"},
}

type Cartridge struct {
	Title      string
	IsColourGB bool
	Type       CartridgeType
	ROMSize    int
	RAMSize    int
	IsJapanese bool
	Filename   string
	MBC        MemoryBankController
}

func NewCartridge(romFile string) (*Cartridge, error) {
	var cart *Cartridge = new(Cartridge)

	rom, err := RetrieveROM(romFile)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error retrieving ROM file: %v", err))
	}

	cart.Filename = romFile
	err = cart.Init(rom)
	if err != nil {
		return nil, err
	}

	return cart, nil
}

func (c *Cartridge) Init(rom []byte) error {
	if size := len(rom); size < 32768 {
		return errors.New(fmt.Sprintf("ROM size %d is too small", size))
	}

	c.Title = strings.TrimSpace(string(rom[0x0134:0x0142]))
	c.IsColourGB = (rom[0x0143] == 0x80) || (rom[0x0143] == 0xC0)

	ctype := rom[0x0147]
	//validate
	if v, ok := CartridgeTypes[ctype]; !ok {
		return errors.New(fmt.Sprintf("Unknown cartridge type: %X for ROM", ctype))
	} else {
		c.Type = v
	}

	if romSize := rom[0x0148]; romSize > 0x06 {
		return errors.New(fmt.Sprintf("Handling for ROM size id: 0x%X is currently unimplemented", romSize))
	} else {
		c.ROMSize = 0x8000 << romSize
	}

	switch rom[0x0149] {
	case 0x00:
		c.RAMSize = 0
	case 0x01:
		c.RAMSize = 2048
	case 0x02:
		c.RAMSize = 8192
	case 0x03:
		c.RAMSize = 32768
	case 0x04:
		c.RAMSize = 131072
	}

	c.IsJapanese = (rom[0x014A] == 0x00)

	switch c.Type.ID {
	case MBC_0:
		c.MBC = NewMBC0(rom)
	case MBC_1, MBC_1_RAM:
		c.MBC = NewMBC1(rom, c.ROMSize, c.RAMSize, false)
	case MBC_1_RAM_BATT:
		c.MBC = NewMBC1(rom, c.ROMSize, c.RAMSize, true)
	case MBC_3_RAM_BATT:
		c.MBC = NewMBC3(rom, c.ROMSize, c.RAMSize, true, false)
	case MBC_3_RAM_BATT_TIMER:
		c.MBC = NewMBC3(rom, c.ROMSize, c.RAMSize, true, true)
	case MBC_5, MBC_5_RAM, MBC_5_RUMBLE, MBC_5_RAM_RUMBLE:
		c.MBC = NewMBC5(rom, c.ROMSize, c.RAMSize, false)
	case MBC_5_RAM_BATT, MBC_5_RAM_BATT_RUMBLE:
		c.MBC = NewMBC5(rom, c.ROMSize, c.RAMSize, true)
	default:
		return errors.New("Error: Cartridge type " + utils.ByteToString(c.Type.ID) + " is currently unsupported")
	}

	return nil
}

func (c *Cartridge) SaveRam(savesDir string) error {
	return c.MBC.SaveRam(savesDir, filepath.Base(c.Filename))
}

func (c *Cartridge) LoadRam(savesDir string) error {
	return c.MBC.LoadRam(savesDir, filepath.Base(c.Filename))
}

func (c *Cartridge) String() string {
	startingString := "Gameboy"
	if c.IsColourGB {
		startingString += " Color"
	}

	var destinationRegion string
	if c.IsJapanese {
		destinationRegion = "Japanese"
	} else {
		destinationRegion = "Non-Japanese"
	}

	var header []string = []string{
		fmt.Sprintf(utils.PadRight("Title:", 19, " ")+"%s", c.Title),
		fmt.Sprintf(utils.PadRight("Type:", 19, " ")+"%s %s", c.Type.Description, utils.ByteToString(c.Type.ID)),
		fmt.Sprintf(utils.PadRight("Destination code:", 19, " ")+"%s", destinationRegion),
		fmt.Sprintf(utils.PadRight("File:", 19, " ")+"%s", c.Filename),
	}

	return fmt.Sprintln("\n"+startingString, "Cartridge") +
		fmt.Sprintln(strings.Repeat("-", 100)) +
		fmt.Sprintln(strings.Join(header, "\n")) +
		fmt.Sprintln(c.MBC) +
		fmt.Sprintln(strings.Repeat("-", 100))
}

func RetrieveROM(filename string) ([]byte, error) {
	file, err := os.Open(filename)

	if err != nil {
		return nil, err
	}
	defer file.Close()

	stats, statsErr := file.Stat()
	if statsErr != nil {
		return nil, statsErr
	}

	var size int64 = stats.Size()
	bytes := make([]byte, size)

	bufr := bufio.NewReader(file)
	_, err = bufr.Read(bytes)

	return bytes, err
}
