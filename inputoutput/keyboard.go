package inputoutput

// #include "keyboard.h"
import "C"

func InitKeyboard() error {
	C.kbd_init()
	//TODO: perform proper error handling in keyboard.c
	return nil
}

func RestoreKeyboard() error {
	C.kbd_restore()
	return nil
}
