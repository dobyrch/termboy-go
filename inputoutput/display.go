package inputoutput

import (
	"syscall"
	"unsafe"
	"github.com/dobyrch/termboy-go/ansi"
	"github.com/dobyrch/termboy-go/types"
)

type Display struct {
        Name                 string
        ScreenSizeMultiplier int
	offX                 int
	offY                 int
}

func (s *Display) init(title string, screenSizeMultiplier int) error {
        //TODO: use ScreenSizeMultiplier as an indicator of whether to use
        //TODO: left half block or top half block
        //TODO: Perhaps use escape code to set title of terminal?
	ansi.HideCursor()
	ansi.ClearScreen()
	//TODO: use constants in ansi.go in place of hex values
        ansi.DefineColor(0x0, 0x000000)
        ansi.DefineColor(0x4, 0x555555)
        ansi.DefineColor(0x6, 0xAAAAAA)
        ansi.DefineColor(0x7, 0xFFFFFF)
	//TODO: how should bright colors be dealt with?
        ansi.DefineColor(0x8, 0x000000)
        ansi.DefineColor(0xC, 0x555555)
        ansi.DefineColor(0xE, 0xAAAAAA)
        ansi.DefineColor(0xF, 0xFFFFFF)

	s.initOffset()

        return nil
}

func (s *Display) drawFrame(screenData *types.Screen) {
        for y := 0; y < SCREEN_HEIGHT; y ++ {
                for x := 0; x < SCREEN_WIDTH; x += 2 {
                        c1 := screenData[y][x]
                        c2 := screenData[y][x+1]

			var fg, bg int

			switch c1.Red {
			case 0:
				fg = 0x0
			case 96:
				fg = 0x4
			case 196:
				fg = 0x6
			case 235:
				fg = 0x7
			}

			switch c2.Red {
			case 0:
				bg = 0x0
			case 96:
				bg = 0x4
			case 196:
				bg = 0x6
			case 235:
				bg = 0x7
			}

			ansi.SetForeground(fg)
			ansi.SetBackground(bg)
			ansi.PutRune('▌', x/2 + s.offX, y + s.offY)
                }
        }
}


func (s *Display) initOffset() {
	var dimensions [4]uint16

	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(0), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&dimensions)), 0, 0, 0); err != 0 {
		return
	}

	x := int(dimensions[1])
	y := int(dimensions[0])

	if (x > 160/2) {
		s.offX = x/2 - 160/4
	}

	if (y > 144) {
		s.offY = y/2 - 144/2
	}
}


func (s *Display) CleanUp() {
	ansi.ClearScreen()
	ansi.ShowCursor()
	ansi.SetForeground(ansi.BLACK)
	ansi.SetBackground(ansi.WHITE)
}
