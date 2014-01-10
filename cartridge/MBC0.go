package cartridge

import (
	"fmt"
	"log"
	"strings"
	"github.com/dobyrch/termboy-go/types"
	"github.com/dobyrch/termboy-go/utils"
)

//Represents ROM only MBC (MBC0)
// - No RAM
// - One ROM bank
type MBC0 struct {
	Name    string
	romBank []byte
}

func NewMBC0(rom []byte) *MBC0 {

	var m *MBC0 = new(MBC0)
	m.Name = "CARTRIDGE-MBC0"
	//ensure only first 32768 bytes are taken
	m.romBank = rom[0x0000:0x8000]

	return m
}

func (m *MBC0) String() string {
	return fmt.Sprintln("\nMemory Bank Controller") +
		fmt.Sprintln(strings.Repeat("-", 50)) +
		fmt.Sprintln(utils.PadRight("ROM Banks:", 18, " "), 1, fmt.Sprintf("(%d bytes)", len(m.romBank)))
}

func (m *MBC0) Write(addr types.Word, value byte) {
//	log.Printf("%s: Attempted to write 0x%X to address %s - this does nothing!", m.Name, value, addr)
}

func (m *MBC0) Read(addr types.Word) byte {
	if addr < 0x0000 || addr > 0x7FFF {
		log.Fatalf(m.Name+": Cannot read from MBC for address: %s!", addr)
	}

	return m.romBank[addr]
}

func (m *MBC0) switchROMBank(bank int) {
	// not needed for MBC0
}

func (m *MBC0) switchRAMBank(bank int) {
	// not needed for MBC0
}

func (m *MBC0) SaveRam(savesDir string, game string) error {
	return nil
}

func (m *MBC0) LoadRam(savesDir string, game string) error {
	return nil
}
