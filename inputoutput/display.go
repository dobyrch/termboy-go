package inputoutput

import (
	"os"
	"fmt"
	"github.com/dobyrch/termboy-go/types"
)

type Display struct {
        Name                 string
        ScreenSizeMultiplier int
        lastFrame            types.Screen
	offX                 int
	offY                 int
}

func (s *Display) init(title string, screenSizeMultiplier int) error {
        //TODO: use ScreenSizeMultiplier as an indicator of whether to use
        //TODO: left half block or top half block
        //TODO: Perhaps use escape code to set title of terminal?
        //TODO: wrap all ansi prints in its own class with methods for each func
        fmt.Printf("%c[?25l", ESC) //Hide the cursor
        fmt.Printf("%c[2J", ESC) //Clear screen

        fmt.Printf("%c]P0000000%c\\", ESC, ESC)
        fmt.Printf("%c]P4555555%c\\", ESC, ESC)
        fmt.Printf("%c]P6AAAAAA%c\\", ESC, ESC)
        fmt.Printf("%c]P7FFFFFF%c\\", ESC, ESC)
        fmt.Printf("%c]P8000000%c\\", ESC, ESC)
        fmt.Printf("%c]PC555555%c\\", ESC, ESC)
        fmt.Printf("%c]PEAAAAAA%c\\", ESC, ESC)
        fmt.Printf("%c]PFFFFFFF%c\\", ESC, ESC)

        return nil
}

func (s *Display) drawFrame(screenData *types.Screen) {
        for y := 0; y < SCREEN_HEIGHT; y ++ {
                for x := 0; x < SCREEN_WIDTH; x += 2 {
                        c1 := screenData[y][x]
                        c2 := screenData[y][x+1]

                        if (s.lastFrame[y][x] != c1 ||
                            s.lastFrame[y][x+1] != c2) {
                                s.lastFrame[y][x] = c1
                                s.lastFrame[y][x+1] = c2

                                var fg, bg int
                                //TODO: in ansi class, set color/bold attr and append codes as needed
                                //TODO: (and define all codes as consts)
                                switch c1.Red {
                                case 0:
                                        fg = 30
                                case 96:
                                        fg = 34
                                case 196:
                                        fg = 36
                                case 235:
                                        fg = 37
                                }

                                switch c2.Red {
                                case 0:
                                        bg = 40
                                case 96:
                                        bg = 44
                                case 196:
                                        bg = 46
                                case 235:
                                        bg = 47
                                }

                                //fmt.Printf("%c[%d;%dH", ESC, y + 1 + s.offY, x/2 + 1 + s.offX)
                                fmt.Printf("%c[%d;%dH", ESC, y + 1, x/2 + 1)
                                //fmt.Printf("%c[%d;%dm%c", ESC, fg, bg, '▀')
                                fmt.Printf("%c[%d;%dm%c", ESC, fg, bg, '▌')
                        }
                }
        }
}

func (s *Display) initOffset() {
	var x, y int

	// Move cursor to bottom right
	fmt.Printf("%c[1000B", ESC)
	fmt.Printf("%c[1000C", ESC)
	fmt.Printf("%c[6n", ESC)

	// Get current position
	cup := fmt.Sprintf("%c[%%d;%%dR", ESC)
	n, scanError := fmt.Scanf(cup, &y, &x)
	file, _ := os.Create("error.out")
	fmt.Fprintf(file, "%d: %s\n", n, scanError)
	file.Close()

	switch {
	case x > 160/2:
		s.offX = (x+1)/2 - 160/4
	case y > 144:
		s.offY = (y+1)/2 - 144/2
	default:
		//TODO: are struct members initialized to 0?
		s.offX = 0
		s.offY = 0
	}
}

func (s *Display) CleanUp() {
        fmt.Printf("%c[?25h", ESC) //Show the cursor
        fmt.Printf("%c[2J", ESC) //Clear screen
        fmt.Printf("%c[H", ESC) //Position cursor in top left
}
