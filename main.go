package main

import (
	"flag"
	"fmt"
	"github.com/dobyrch/termboy-go/apu"
	"github.com/dobyrch/termboy-go/cartridge"
	"github.com/dobyrch/termboy-go/cpu"
	"github.com/dobyrch/termboy-go/gpu"
	"github.com/dobyrch/termboy-go/inputoutput"
	"github.com/dobyrch/termboy-go/mmu"
	"github.com/dobyrch/termboy-go/timer"
	"github.com/dobyrch/termboy-go/types"
	//TODO: ensure that all fatal logs run Poweroff!
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

const FRAME_CYCLES = 70224
const TITLE string = "termboy"

var VERSION string

type GameBoy struct {
	gpu   *gpu.GPU
	cpu   *cpu.CPU
	mmu   *mmu.MMU
	io    *inputoutput.IO
	apu   *apu.APU
	timer *timer.Timer
	//debugOptions           *DebugOptions
	config                 Config
	cpuClockAcc            int
	frameCount             int
	stepCount              int
	inBootMode             bool
	fpsSamples             []int
	averageFramesPerSecond int
}

func NewGameBoy() *GameBoy {
	gb := new(GameBoy)

	gb.mmu = mmu.NewMMU()
	gb.cpu = cpu.NewCPU()
	gb.cpu.LinkMMU(gb.mmu)

	gb.io = inputoutput.NewIO()
	gb.gpu = gpu.NewGPU()
	gb.apu = apu.NewAPU()
	gb.timer = timer.NewTimer()

	//mmu will process interrupt requests from GPU (i.e. it will set appropriate flags)
	gb.gpu.LinkIRQHandler(gb.mmu)
	gb.timer.LinkIRQHandler(gb.mmu)
	gb.io.KeyHandler.LinkIRQHandler(gb.mmu)

	gb.mmu.ConnectPeripheral(gb.apu, 0xFF10, 0xFF3F)
	gb.mmu.ConnectPeripheral(gb.gpu, 0x8000, 0x9FFF)
	gb.mmu.ConnectPeripheral(gb.gpu, 0xFE00, 0xFE9F)
	gb.mmu.ConnectPeripheral(gb.gpu, 0xFF57, 0xFF6F)
	gb.mmu.ConnectPeripheralOn(gb.gpu, 0xFF40, 0xFF41, 0xFF42, 0xFF43, 0xFF44, 0xFF45, 0xFF47, 0xFF48, 0xFF49, 0xFF4A, 0xFF4B, 0xFF4F)
	gb.mmu.ConnectPeripheralOn(gb.io.KeyHandler, 0xFF00)
	gb.mmu.ConnectPeripheralOn(gb.timer, 0xFF04, 0xFF05, 0xFF06, 0xFF07)

	return gb
}

func (gb *GameBoy) DoFrame() {
	for gb.cpuClockAcc < FRAME_CYCLES {
		/*if gb.debugOptions.debuggerOn && gb.cpu.PC == gb.debugOptions.breakWhen {
			gb.Pause()
		}*/

		if gb.config.DumpState && !gb.cpu.Halted {
			log.Println(gb.cpu)
		}
		gb.Step()
	}
}

func (gb *GameBoy) Step() {
	cycles := gb.cpu.Step()
	//GPU is unaffected by CPU speed changes
	gb.gpu.Step(cycles)
	gb.cpuClockAcc += cycles

	//these are affected by CPU speed changes
	gb.timer.Step(cycles / gb.cpu.Speed)

	gb.stepCount++
	//value in FF50 means gameboy has finished booting
	if gb.inBootMode {
		if gb.mmu.ReadByte(0xFF50) != 0x00 {
			gb.cpu.PC = 0x0100
			gb.mmu.SetInBootMode(false)
			gb.inBootMode = false

			//put the GPU in color mode if cartridge is ColorGB and user has specified color GB mode
			gb.SetHardwareMode(gb.config.ColorMode)
			log.Println("Finished GB boot program, launching game...")
		}
	}
}

func (gb *GameBoy) Run() {
	currentTime := time.Now()
	for {
		gb.DoFrame()
		gb.cpuClockAcc = 0
		gb.frameCount++
		if gb.config.DisplayFPS {
			if time.Since(currentTime) >= (1 * time.Second) {
				gb.StoreFPSSample(gb.frameCount / 1.0)
				log.Println("Average frames per second:", gb.averageFramesPerSecond)
				currentTime = time.Now()
				gb.frameCount = 0
			}
		}
	}
}

func (gb *GameBoy) StoreFPSSample(sample int) {
	gb.fpsSamples = append(gb.fpsSamples, sample)
	if len(gb.fpsSamples) == 5 {
		average := 0
		for _, i := range gb.fpsSamples {
			average += i
		}
		gb.averageFramesPerSecond = average / 5
		gb.fpsSamples = gb.fpsSamples[1:]
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Usage = PrintHelp
	flag.Parse()

	if *help {
		PrintHelp()
		os.Exit(1)
	}

	if os.Getenv("TERM") != "linux" {
		fmt.Println("Term Boy can only be run in the Linux console")
		fmt.Println("(Try pressing CTRL+ALT+F2)")
		os.Exit(1)
	}

	if flag.NArg() != 1 {
		fmt.Println("Please specify the location of a ROM to boot")
		os.Exit(1)
	}

	//Parse and validate settings file (if found)
	conf := NewConfig()

	if err := conf.ConfigureSettingsDirectory(); err != nil {
		log.Fatalf("Error configuring settings directory: %v", err)
	}

	if err := conf.LoadConfig(); err != nil {
		log.Fatalf("Error encountered attempting to load configuration file: %v", err)
	}

	//command line flags take precedence
	conf.OverrideConfigWithAnySetFlags()

	log.Println(conf)
	romFilename := flag.Arg(0)

	cart, err := cartridge.NewCartridge(romFilename)
	if err != nil {
		log.Fatal(err)
	}

	var gb *GameBoy = NewGameBoy()
	defer gb.Poweroff()

	gb.config = *conf

	if err := gb.mmu.LoadBIOS(BOOTROM); err != nil {
		log.Fatal(err)
	}

	gb.mmu.LoadCartridge(cart)
	//gb.debugOptions = new(DebugOptions)
	//gb.debugOptions.Init(gb.config.DumpState)

	/*if gb.config.Debug {
		log.Println("Emulator will start in debug mode")
		gb.debugOptions.debuggerOn = true

		//set breakpoint if defined
		if b, err := utils.StringToWord(gb.config.BreakOn); err != nil {
			log.Panicln("Cannot parse breakpoint:", gb.config.BreakOn, "\n\t", err)
		} else {
			gb.debugOptions.breakWhen = types.Word(b)
			log.Println("Emulator will break into debugger when PC = ", gb.debugOptions.breakWhen)
		}
	}*/

	//append cartridge name and filename to title
	gb.config.Title += fmt.Sprintf(" - %s - %s", filepath.Base(cart.Filename), cart.Title)

	gb.io.Init()

	//load RAM into MBC (if supported)
	gb.mmu.LoadCartridgeRam(gb.config.SavesDir)
	gb.gpu.LinkScreen(gb.io.ScreenOutputChannel)
	gb.setupBoot()

	log.Println("Completed setup")

	log.Println("Starting emulator")

	//TODO: Move signal handling to a more appropriate place (inputoutput)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-sigc
		gb.Poweroff()
	}()

	//Start emulator code in a goroutine
	go gb.Run()

	//lock the OS thread here
	runtime.LockOSThread()

	//Run IO operations until user presses ESC
	gb.io.Run()

}

func (gb *GameBoy) setupBoot() {
	if gb.config.SkipBoot {
		log.Println("Boot sequence disabled")
		gb.setupWithoutBoot()
	} else {
		log.Println("Boot sequence enabled")
		gb.setupWithBoot()
	}
}

func (gb *GameBoy) setupWithBoot() {
	gb.inBootMode = true
	gb.mmu.WriteByte(0xFF50, 0x00)
}

//Determine if ColorGB hardware should be enabled
func (gb *GameBoy) SetHardwareMode(isColor bool) {
	if isColor {
		gb.cpu.R.A = 0x11
		gb.gpu.RunningColorGBHardware = gb.mmu.IsCartridgeColor()
		gb.mmu.RunningColorGBHardware = true
	} else {
		gb.cpu.R.A = 0x01
		gb.gpu.RunningColorGBHardware = false
		gb.mmu.RunningColorGBHardware = false
	}
}

func (gb *GameBoy) setupWithoutBoot() {
	gb.inBootMode = false
	gb.mmu.SetInBootMode(false)
	gb.cpu.PC = 0x100
	gb.SetHardwareMode(gb.config.ColorMode)
	gb.cpu.R.F = 0xB0
	gb.cpu.R.B = 0x00
	gb.cpu.R.C = 0x13
	gb.cpu.R.D = 0x00
	gb.cpu.R.E = 0xD8
	gb.cpu.R.H = 0x01
	gb.cpu.R.L = 0x4D
	gb.cpu.SP = 0xFFFE
	gb.mmu.WriteByte(0xFF05, 0x00)
	gb.mmu.WriteByte(0xFF06, 0x00)
	gb.mmu.WriteByte(0xFF07, 0x00)
	gb.mmu.WriteByte(0xFF10, 0x80)
	gb.mmu.WriteByte(0xFF11, 0xBF)
	gb.mmu.WriteByte(0xFF12, 0xF3)
	gb.mmu.WriteByte(0xFF14, 0xBF)
	gb.mmu.WriteByte(0xFF16, 0x3F)
	gb.mmu.WriteByte(0xFF17, 0x00)
	gb.mmu.WriteByte(0xFF19, 0xBF)
	gb.mmu.WriteByte(0xFF1A, 0x7F)
	gb.mmu.WriteByte(0xFF1B, 0xFF)
	gb.mmu.WriteByte(0xFF1C, 0x9F)
	gb.mmu.WriteByte(0xFF1E, 0xBF)
	gb.mmu.WriteByte(0xFF20, 0xFF)
	gb.mmu.WriteByte(0xFF21, 0x00)
	gb.mmu.WriteByte(0xFF22, 0x00)
	gb.mmu.WriteByte(0xFF23, 0xBF)
	gb.mmu.WriteByte(0xFF24, 0x77)
	gb.mmu.WriteByte(0xFF25, 0xF3)
	gb.mmu.WriteByte(0xFF26, 0xF1)
	gb.mmu.WriteByte(0xFF40, 0x91)
	gb.mmu.WriteByte(0xFF42, 0x00)
	gb.mmu.WriteByte(0xFF43, 0x00)
	gb.mmu.WriteByte(0xFF45, 0x00)
	gb.mmu.WriteByte(0xFF47, 0xFC)
	gb.mmu.WriteByte(0xFF48, 0xFF)
	gb.mmu.WriteByte(0xFF49, 0xFF)
	gb.mmu.WriteByte(0xFF4A, 0x00)
	gb.mmu.WriteByte(0xFF4B, 0x00)
	gb.mmu.WriteByte(0xFF50, 0x00)
	gb.mmu.WriteByte(0xFFFF, 0x00)
}

func (gb *GameBoy) Poweroff() {
	gb.mmu.SaveCartridgeRam(gb.config.SavesDir)
	gb.io.Display.CleanUp()
	gb.io.KeyHandler.RestoreKeyboard()

	if r := recover(); r != nil {
		fmt.Println(r)
	}

	os.Exit(0)
}

/*func (gb *GameBoy) Pause() {
	log.Println("DEBUGGER: Breaking because PC ==", gb.debugOptions.breakWhen)
	b := bufio.NewWriter(os.Stdout)
	r := bufio.NewReader(os.Stdin)

	log.Fprintln(b, "Debug mode, type ? for help")
	for gb.debugOptions.debuggerOn {
		var instruction string
		b.Flush()
		log.Fprint(b, "> ")
		b.Flush()
		instruction, _ = r.ReadString('\n')
		b.Flush()
		var instructions []string = strings.Split(strings.Replace(instruction, "\n", "", -1), " ")
		b.Flush()

		command := instructions[0]

		if command == "c" {
			break
		}

		//dispatch
		if v, ok := gb.debugOptions.debugFuncMap[command]; ok {
			v(gb, instructions[1:]...)
		} else {
			log.Fprintln(b, "Unknown command:", command)
			log.Fprintln(b, "Debug mode, type ? for help")
		}
	}
}*/

func (gb *GameBoy) Reset() {
	log.Println("Resetting system")
	gb.cpu.Reset()
	gb.gpu.Reset()
	gb.mmu.Reset()
	gb.apu.Reset()
	gb.io.KeyHandler.Reset()
	gb.io.ScreenOutputChannel <- &(types.Screen{})
	gb.setupBoot()
}

var BOOTROM []byte = []byte{
	0x31, 0xFE, 0xFF, 0xAF, 0x21, 0xFF, 0x9F, 0x32, 0xCB, 0x7C, 0x20, 0xFB, 0x21, 0x26, 0xFF, 0x0E,
	0x11, 0x3E, 0x80, 0x32, 0xE2, 0x0C, 0x3E, 0xF3, 0xE2, 0x32, 0x3E, 0x77, 0x77, 0x3E, 0xFC, 0xE0,
	0x47, 0x11, 0xa8, 0x00, 0x21, 0x10, 0x80, 0x1A, 0xCD, 0x95, 0x00, 0xCD, 0x96, 0x00, 0x13, 0x7B,
	0xFE, 0x34, 0x20, 0xF3, 0x11, 0xD8, 0x00, 0x06, 0x08, 0x1A, 0x13, 0x22, 0x23, 0x05, 0x20, 0xF9,
	0x3E, 0x19, 0xEA, 0x10, 0x99, 0x21, 0x2F, 0x99, 0x0E, 0x0C, 0x3D, 0x28, 0x08, 0x32, 0x0D, 0x20,
	0xF9, 0x2E, 0x0F, 0x18, 0xF3, 0x67, 0x3E, 0x64, 0x57, 0xE0, 0x42, 0x3E, 0x91, 0xE0, 0x40, 0x04,
	0x1E, 0x02, 0x0E, 0x0C, 0xF0, 0x44, 0xFE, 0x90, 0x20, 0xFA, 0x0D, 0x20, 0xF7, 0x1D, 0x20, 0xF2,
	0x0E, 0x13, 0x24, 0x7C, 0x1E, 0x83, 0xFE, 0x62, 0x28, 0x06, 0x1E, 0xC1, 0xFE, 0x64, 0x20, 0x06,
	0x7B, 0xE2, 0x0C, 0x3E, 0x87, 0xE2, 0xF0, 0x42, 0x90, 0xE0, 0x42, 0x15, 0x20, 0xD2, 0x05, 0x20,
	0x4F, 0x16, 0x20, 0x18, 0xCB, 0x4F, 0x06, 0x04, 0xC5, 0xCB, 0x11, 0x17, 0xC1, 0xCB, 0x11, 0x17,
	0x05, 0x20, 0xF5, 0x22, 0x23, 0x22, 0x23, 0xC9, 0xFF, 0x33, 0xCC, 0x03, 0x00, 0x0C, 0x00, 0x0D,
	0x00, 0x0B, 0x00, 0x06, 0x00, 0x0C, 0xFC, 0xCF, 0x8C, 0xC8, 0x00, 0x0F, 0x00, 0x03, 0x00, 0x03,
	0x33, 0x33, 0x67, 0x63, 0x6E, 0x0E, 0xEC, 0xCC, 0x33, 0x33, 0xB2, 0x22, 0x66, 0x66, 0xCC, 0xCF,
	0xDD, 0xD8, 0x99, 0x9F, 0xB9, 0x80, 0x3E, 0xCC, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x21, 0x04, 0x01, 0x11, 0xA8, 0x00, 0x1A, 0x13, 0xBE, 0x20, 0x01, 0x23, 0x7D, 0xFE, 0x34, 0x20,
	0xF5, 0x06, 0x19, 0x78, 0x86, 0x23, 0x05, 0x20, 0xFB, 0x86, 0x20, 0x01, 0x3E, 0x01, 0xE0, 0x50,
}
