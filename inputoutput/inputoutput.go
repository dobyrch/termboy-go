package inputoutput

import (
	"github.com/dobyrch/termboy-go/components"
	"github.com/dobyrch/termboy-go/constants"
	"github.com/dobyrch/termboy-go/types"
	"fmt"
	"os"
	"io"
	"log"
)

const PREFIX string = "IO"
const ESC byte = 0x1B
const ROW_1 byte = 0x10
const ROW_2 byte = 0x20
const SCREEN_WIDTH int = 160
const SCREEN_HEIGHT int = 144

//TODO: move control related stuff to keyboard.go?
//TODO: use var names in struct initializer for improved legibility
var DefaultControlScheme ControlScheme = ControlScheme{0x21, 0x1F, 0x12, 0x20, 0x25, 0x24, 0x22, 0x23}

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
}

func (k *KeyHandler) Name() string {
	return PREFIX + "-KEYB"
}

func (k *KeyHandler) Reset() {
	log.Printf("%s: Resetting", k.Name())
	k.rows[0], k.rows[1] = 0x0F, 0x0F
	k.colSelect = 0x00
}

func (k *KeyHandler) LinkIRQHandler(m components.IRQHandler) {
	k.irqHandler = m
	log.Printf("%s: Linked IRQ Handler to Keyboard Handler", k.Name())
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
	case 0x1:
		//TODO: called single function for signals and intentional exits
		//TODO: clean up screen on exit
		RestoreKeyboard()
		log.Fatal("Thanks for playing!")
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
	if key & (1 << 7) != 0{
		k.KeyUp(key &^ (1 << 7))
	} else {
		k.KeyDown(key)
	}
}

type IO struct {
	KeyHandler           *KeyHandler
	Display              *Display
	ScreenOutputChannel  chan *types.Screen
	AudioOutputChannel   chan int
	KeyboardInputChannel chan byte
}

func NewIO() *IO {
	var i *IO = new(IO)
	i.KeyHandler = new(KeyHandler)
	i.Display = new(Display)
	i.ScreenOutputChannel = make(chan *types.Screen)
	i.AudioOutputChannel = make(chan int)
	i.KeyboardInputChannel = make(chan byte)
	return i
}

func (i *IO) Init(title string, screenSize int, onCloseHandler func()) error {
	//TODO: call onCloseHandler in gbc upon receiving signal
	//TODO: put RestoreKeyboard in onCloseHandler if appropriate

	err := i.Display.init(title, screenSize)
	//TODO: Is it necesssary to return an error?
	if err != nil {
		return err
	}

	i.KeyHandler.Init(DefaultControlScheme) //TODO: allow user to define controlscheme

	return nil
}

//This will wait for updates to the display or audio and dispatch them accordingly
func (i *IO) Run() {
	go i.pollStdin()
	for {
		select {
		case data := <-i.ScreenOutputChannel:
			i.Display.drawFrame(data)
		case data := <-i.AudioOutputChannel:
			log.Println("Writing %d to audio!", data)
		case data := <-i.KeyboardInputChannel:
			i.KeyHandler.keyEvent(data)
		}
	}
}

func (i *IO) pollStdin() {
	var b = make([]byte, 1)

	for {
		_, err := io.ReadFull(os.Stdin, b)
		if (err == nil) {
			i.KeyboardInputChannel <-b[0]
		}
	}
}

type coord struct {
	x int
	y int
}

type Display struct {
	Name                 string
	ScreenSizeMultiplier int
	Screen map[coord]byte
}

func (s *Display) init(title string, screenSizeMultiplier int) error {
	s.Screen = make(map[coord]byte)
	//TODO: use ScreenSizeMultiplier as an indicator of whether to use
	//TODO: left half block or top half block
	//TODO: Perhaps use escape code to set title of terminal?

	//TODO: wrap all ansii prints in its own class with methods for each func
        //TODO: show the cursor after termination
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
				//TODO: have a 'big' and 'small' mode (top/left half)
				//TODO: check if setfont is available on BSD and can change height
				fmt.Printf("%c[%d;%dm%c", ESC, fg, bg, '▀')
			}
                }
        }
}
