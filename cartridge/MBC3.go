package cartridge

import (
	"fmt"
	"github.com/dobyrch/termboy-go/types"
	"github.com/dobyrch/termboy-go/utils"
	"strings"
)

//Represents MBC3
type MBC3 struct {
	Name            string
	romBank0        []byte
	romBanks        [][]byte
	ramBanks        [][]byte
	selectedROMBank int
	selectedRAMBank int
	hasRAM          bool
	ramEnabled      bool
	ROMSize         int
	RAMSize         int
	hasBattery      bool
}

func NewMBC3(rom []byte, romSize int, ramSize int, hasBattery bool) *MBC3 {
	var m *MBC3 = new(MBC3)

	m.Name = "CARTRIDGE-MBC3"
	m.hasBattery = hasBattery
	m.ROMSize = romSize
	m.RAMSize = ramSize

	if ramSize > 0 {
		m.hasRAM = true
		m.ramEnabled = true
		m.selectedRAMBank = 0
		m.ramBanks = populateRAMBanks(4)
	}

	m.selectedROMBank = 0
	m.romBank0 = rom[0x0000:0x4000]
	m.romBanks = populateROMBanks(rom, m.ROMSize/0x4000)

	return m
}

func (m *MBC3) String() string {
	var batteryStr string
	if m.hasBattery {
		batteryStr += "Yes"
	} else {
		batteryStr += "No"
	}

	return fmt.Sprintln("\nMemory Bank Controller") +
		fmt.Sprintln(strings.Repeat("-", 50)) +
		fmt.Sprintln(utils.PadRight("ROM Banks:", 18, " "), len(m.romBanks), fmt.Sprintf("(%d bytes)", m.ROMSize)) +
		fmt.Sprintln(utils.PadRight("RAM Banks:", 18, " "), m.RAMSize/0x2000, fmt.Sprintf("(%d bytes)", m.RAMSize)) +
		fmt.Sprintln(utils.PadRight("Battery:", 18, " "), batteryStr)
}

func (m *MBC3) Write(addr types.Word, value byte) {
	switch {
	case addr >= 0x0000 && addr <= 0x1FFF:
		if m.hasRAM {
			if r := value & 0x0F; r == 0x0A {
				m.ramEnabled = true
			} else {
				m.ramEnabled = false
			}
		}
	case addr >= 0x2000 && addr <= 0x3FFF:
		m.switchROMBank(int(value & 0x7F)) //7 bits rather than 5
	case addr >= 0x4000 && addr <= 0x5FFF:
		m.switchRAMBank(int(value & 0x03))
	//case addr >= 0x6000 && addr <= 0x7FFF:
	// RTC stuff....
	//return
	case addr >= 0xA000 && addr <= 0xBFFF:
		if m.hasRAM && m.ramEnabled {
			m.ramBanks[m.selectedRAMBank][addr-0xA000] = value
		}
	}
}

func (m *MBC3) Read(addr types.Word) byte {
	//ROM Bank 0
	if addr < 0x4000 {
		return m.romBank0[addr]
	}

	//Switchable ROM BANK
	if addr >= 0x4000 && addr < 0x8000 {
		return m.romBanks[m.selectedROMBank][addr-0x4000]
	}

	//Upper bounds of memory map.
	if addr >= 0xA000 && addr <= 0xC000 {
		if m.hasRAM && m.ramEnabled {
			return m.ramBanks[m.selectedRAMBank][addr-0xA000]
		}
	}

	return 0x00
}

func (m *MBC3) switchROMBank(bank int) {
	m.selectedROMBank = bank
}

func (m *MBC3) switchRAMBank(bank int) {
	m.selectedRAMBank = bank
}

func (m *MBC3) SaveRam(savesDir string, game string) error {
	if m.hasRAM && m.hasBattery {
		s := NewSaveFile(savesDir, game)
		err := s.Save(m.ramBanks)
		s = nil
		return err
	}
	return nil
}

func (m *MBC3) LoadRam(savesDir string, game string) error {
	if m.hasRAM && m.hasBattery {
		s := NewSaveFile(savesDir, game)
		banks, err := s.Load(4)
		if err != nil {
			return err
		}
		m.ramBanks = banks
		s = nil
	}
	return nil
}
