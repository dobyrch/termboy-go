/**
 * Just a place holder for now
 * User: danielharper
 * Date: 28/02/2013
 * Time: 21:26
 */
package apu

import (
	//"log"
	"github.com/dobyrch/termboy-go/components"
	"github.com/dobyrch/termboy-go/types"
)

type APU struct {
	mem [0x41]byte
}

func NewAPU() *APU {
	var a *APU = new(APU)
	a.Reset()
	return a
}

func (apu *APU) Name() string {
	return "APU"
}

func (apu *APU) Read(addr types.Word) byte {
	if addr == 0xFF26 {
		return 0x00
	}
	return apu.mem[addr-0xFF00]
}

func (apu *APU) Write(addr types.Word, value byte) {
	apu.mem[addr-0xFF00] = value
}

func (apu *APU) LinkIRQHandler(m components.IRQHandler) {

}

func (apu *APU) Reset() {
//	log.Println(apu.Name()+": Resetting", apu.Name())
}
