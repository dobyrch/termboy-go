package inputoutput

import (
	"os/exec"
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

func (s *Display) init() {
        if err := exec.Command("setfont", "-h4").Run(); err != nil {
                panic("Failed to set font height")
        }

	ansi.HideCursor()
	ansi.ClearScreen()
        ansi.DefineColor(ansi.BLACK, 0x000000)
        ansi.DefineColor(ansi.BLUE, 0x555555)
        ansi.DefineColor(ansi.CYAN, 0xAAAAAA)
        ansi.DefineColor(ansi.WHITE, 0xFFFFFF)
	s.initOffset()
}

func (s *Display) drawFrame(screenData *types.Screen) {
        for y := 0; y < SCREEN_HEIGHT; y++ {
                for x := 0; x < SCREEN_WIDTH; x++ {
                        c := screenData[y][x]

			var bg int

			switch c.Red {
			case 0:
				bg = ansi.BLACK
			case 96:
				bg = ansi.BLUE
			case 196:
				bg = ansi.CYAN
			case 235:
				bg = ansi.WHITE
			}

			ansi.SetBackground(bg)
			ansi.PutRune(' ', x + s.offX, y + s.offY)
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

	if (x > 160) {
		s.offX = x/2 - 160/2
	}

	if (y > 144) {
		s.offY = y/2 - 144/2
	}
}


func (s *Display) CleanUp() {
	ansi.ClearScreen()
	ansi.ShowCursor()
	ansi.SetForeground(ansi.BLACK)
	ansi.SetBackground(ansi.BRIGHT | ansi.WHITE)
	exec.Command("setfont").Run()
}
