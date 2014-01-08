package cartridge

import "github.com/dobyrch/termboy-go/types"

type MemoryBankController interface {
	Write(addr types.Word, value byte)
	Read(addr types.Word) byte
	SaveRam(savesDir string, game string) error
	LoadRam(savesDir string, game string) error
	switchROMBank(bank int)
	switchRAMBank(bank int)
}

func populateROMBanks(rom []byte, noOfBanks int) [][]byte {
	romBanks := make([][]byte, noOfBanks)

	//ROM Bank 0 and 1 are the same
	romBanks[0] = rom[0x4000:0x8000]
	var chunk int = 0x4000
	for i := 1; i < noOfBanks; i++ {
		romBanks[i] = rom[chunk : chunk+0x4000]
		chunk += 0x4000
	}

	return romBanks
}

func populateRAMBanks(noOfBanks int) [][]byte {
	ramBanks := make([][]byte, noOfBanks)

	for i := 0; i < noOfBanks; i++ {
		ramBanks[i] = make([]byte, 0x2000)
	}

	return ramBanks
}
