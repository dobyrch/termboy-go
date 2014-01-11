package inputoutput

import (
	"io"
	//"log"
	"os"
	"github.com/dobyrch/termboy-go/types"
)

const PREFIX string = "IO"
const ROW_1 byte = 0x10
const ROW_2 byte = 0x20
const SCREEN_WIDTH int = 160
const SCREEN_HEIGHT int = 144

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

func (i *IO) Init(title string, screenSize int) error {
	err := i.Display.init(title, screenSize)
	//TODO: Is it necesssary to return an error?
	if err != nil {
		return err
	}

	i.KeyHandler.Init(DefaultControlScheme) //TODO: allow user to define controlscheme
	go i.pollStdin()

	return nil
}

//This will wait for updates to the display or audio and dispatch them accordingly
func (i *IO) Run() {
	for {
		select {
		case data := <-i.ScreenOutputChannel:
			i.Display.drawFrame(data)
		//case data := <-i.AudioOutputChannel:
//			log.Println("Writing %d to audio!", data)
		case data := <-i.KeyboardInputChannel:
			if (data == 0x1) {
				return //Stop if ESC pressed
			} else {
				i.KeyHandler.keyEvent(data)
			}
		}
	}
}

func (i *IO) pollStdin() {
        var b = make([]byte, 1)

        for {
                _, err := io.ReadFull(os.Stdin, b)
                if (err == nil) {
			i.KeyboardInputChannel <- b[0]
                }
        }
}
