package inputoutput

import (
	// #include "keyboard.h"
	"C"
	//"log"
	"github.com/dobyrch/termboy-go/components"
	"github.com/dobyrch/termboy-go/constants"
	"github.com/dobyrch/termboy-go/types"
)

/*func InitKeyboard() error {
	C.kbd_init()
	//TODO: perform proper error handling in keyboard.c
	return nil
}*/

var DefaultControlScheme ControlScheme = ControlScheme{
        RIGHT:  0x21,
        LEFT:   0x1F,
        UP:     0x12,
        DOWN:   0x20,
        A:      0x25,
        B:      0x24,
        SELECT: 0x22,
        START:  0x23,
}

type ControlScheme struct {
        RIGHT  byte
        LEFT   byte
        UP     byte
        DOWN   byte
        A      byte
        B      byte
        SELECT byte
        START  byte
}

type KeyHandler struct {
        controlScheme ControlScheme
        colSelect     byte
        rows          [2]byte
        irqHandler    components.IRQHandler
}

func (k *KeyHandler) Init(cs ControlScheme) {
        k.controlScheme = cs
        k.Reset()
	C.kbd_init()
}

func (k *KeyHandler) Name() string {
        return PREFIX + "-KEYB"
}

func (k *KeyHandler) Reset() {
//        log.Printf("%s: Resetting", k.Name())
        k.rows[0], k.rows[1] = 0x0F, 0x0F
        k.colSelect = 0x00
}

func (k *KeyHandler) LinkIRQHandler(m components.IRQHandler) {
        k.irqHandler = m
//        log.Printf("%s: Linked IRQ Handler to Keyboard Handler", k.Name())
}

func (k *KeyHandler) Read(addr types.Word) byte {
        var value byte

        switch k.colSelect {
        case ROW_1:
                value = k.rows[1]
        case ROW_2:
                value = k.rows[0]
        default:
                value = 0x00
        }

        return value
}

func (k *KeyHandler) Write(addr types.Word, value byte) {
        k.colSelect = value & 0x30
}

//released sets bit for key to 0
func (k *KeyHandler) KeyDown(key byte) {
        k.irqHandler.RequestInterrupt(constants.JOYP_HILO_IRQ)
        switch key {
        case k.controlScheme.RIGHT:
                k.rows[0] &^= 0x1
        case k.controlScheme.LEFT:
                k.rows[0] &^= 0x2
        case k.controlScheme.UP:
                k.rows[0] &^= 0x4
        case k.controlScheme.DOWN:
                k.rows[0] &^= 0x8
        case k.controlScheme.A:
                k.rows[1] &^= 0x1
        case k.controlScheme.B:
                k.rows[1] &^= 0x2
        case k.controlScheme.SELECT:
                k.rows[1] &^= 0x4
        case k.controlScheme.START:
                k.rows[1] &^= 0x8
        }
}

//released sets bit for key to 1
func (k *KeyHandler) KeyUp(key byte) {
        switch key {
        case k.controlScheme.RIGHT:
                k.rows[0] |= 0x1
        case k.controlScheme.LEFT:
                k.rows[0] |= 0x2
        case k.controlScheme.UP:
                k.rows[0] |= 0x4
        case k.controlScheme.DOWN:
                k.rows[0] |= 0x8
        case k.controlScheme.A:
                k.rows[1] |= 0x1
        case k.controlScheme.B:
                k.rows[1] |= 0x2
        case k.controlScheme.SELECT:
                k.rows[1] |= 0x4
        case k.controlScheme.START:
                k.rows[1] |= 0x8
        }
}

func (k *KeyHandler) keyEvent(key byte) {
        if key & (1 << 7) != 0 {
                k.KeyUp(key &^ (1 << 7))
        } else {
                k.KeyDown(key)
        }
}

func (k *KeyHandler) RestoreKeyboard() error {
	C.kbd_restore()
	return nil
}
