package constants

import "github.com/dobyrch/termboy-go/types"

//interrupt handler addresses
const (
	V_BLANK_IR_ADDR        byte = 0x40
	LCD_IR_ADDR                 = 0x48
	TIMER_OVERFLOW_IR_ADDR      = 0x50
	JOYP_HILO_IR_ADDR           = 0x60
)

const (
	V_BLANK_IRQ        byte = 0x01 //bit 0
	LCD_IRQ                 = 0x02 //bit 1
	TIMER_OVERFLOW_IRQ      = 0x04 // bit 2
	JOYP_HILO_IRQ           = 0x10 //bit 4
)

const (
	SIXTEENMB_ROM_8KBRAM = iota
	FOURMB_ROM_32KBRAM
)

//memory addresses
const (
	INTERRUPT_ENABLED_FLAG_ADDR types.Word = 0xFFFF
	INTERRUPT_FLAG_ADDR                    = 0xFF0F
)
