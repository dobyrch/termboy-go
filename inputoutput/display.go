package inputoutput

import (
	"fmt"
	"github.com/dobyrch/termboy-go/types"
)

type coord struct {
        x int
        y int
}

type Display struct {
        Name                 string
        ScreenSizeMultiplier int
        //TODO: try using array instead
        Screen map[coord]byte
}

func (s *Display) init(title string, screenSizeMultiplier int) error {
        s.Screen = make(map[coord]byte)
        //TODO: use ScreenSizeMultiplier as an indicator of whether to use
        //TODO: left half block or top half block
        //TODO: Perhaps use escape code to set title of terminal?

        //TODO: wrap all ansii prints in its own class with methods for each func
        fmt.Printf("%c[?25l", ESC) //Hide the cursor
        fmt.Printf("%c[2J", ESC) //Clear screen
        fmt.Printf("%c[H", ESC) //Position cursor in top left

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
        for y := 0; y < SCREEN_HEIGHT; y += 2 {
                for x := 0; x < SCREEN_WIDTH; x++ {
                        c1 := screenData[y][x].Red
                        c2 := screenData[y+1][x].Red

                        if (s.Screen[coord{x, y}] != c1 ||
                            s.Screen[coord{x, y+1}] != c2) {
                                s.Screen[coord{x, y}] = c1
                                s.Screen[coord{x, y+1}] = c2
                                var fg, bg int

                                //TODO: in ansii class, set color/bold attr and append codes as needed
                                //TODO: (and define all codes as consts)
                                switch c1 {
                                case 0:
                                        fg = 30
                                case 96:
                                        fg = 34
                                case 196:
                                        fg = 36
                                case 235:
                                        fg = 37
                                }

                                switch c2 {
                                case 0:
                                        bg = 40
                                case 96:
                                        bg = 44
                                case 196:
                                        bg = 46
                                case 235:
                                        bg = 47
                                }

                                fmt.Printf("%c[%d;%dH", ESC, y/2 + 1, x + 1)
                                fmt.Printf("%c[%d;%dm%c", ESC, fg, bg, '▀')
                        }
                }
        }
}

func (s *Display) CleanUp() {
        fmt.Printf("%c[?25h", ESC) //Show the cursor
        fmt.Printf("%c[2J", ESC) //Clear screen
        fmt.Printf("%c[H", ESC) //Position cursor in top left
}
