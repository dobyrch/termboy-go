package main

//Probably needlessly complex debug facility, but useful none the less

/*
import (
	//"code.google.com/p/freetype-go/freetype"
	//"code.google.com/p/freetype-go/freetype/truetype"
	"errors"
	"fmt"
	"github.com/dobyrch/termboy-go/gpu"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"github.com/dobyrch/termboy-go/types"
	"github.com/dobyrch/termboy-go/utils"
)

type DebugCommandHandler func(*GomeboyColor, ...string)

type DebugOptions struct {
	debuggerOn   bool
	breakWhen    types.Word
	watches      map[types.Word]byte
	debugFuncMap map[string]DebugCommandHandler
	debugHelpStr []string
	stepDump     bool
}

func (g *DebugOptions) help() {
	//fmt.Println("Commands are: -")
	for _, desc := range g.debugHelpStr {
		//fmt.Println("	-", desc)
	}
}

func (g *DebugOptions) Init(cpuDumpOnStep bool) {
	g.debuggerOn = false
	g.debugFuncMap = make(map[string]DebugCommandHandler)
	g.watches = make(map[types.Word]byte)
	g.stepDump = cpuDumpOnStep
	g.AddDebugFunc("p", "Print CPU state", func(gbc *GomeboyColor, remaining ...string) {
		//fmt.Println(gbc.cpu)
	})

	g.AddDebugFunc("r", "Reset", func(gbc *GomeboyColor, remaining ...string) {
		gbc.Reset()
	})

	g.AddDebugFunc("?", "Print this help message", func(gbc *GomeboyColor, remaining ...string) {
		g.help()
	})

	g.AddDebugFunc("help", "Print this help message", func(gbc *GomeboyColor, remaining ...string) {
		g.help()
	})

	g.AddDebugFunc("d", "Disconnect from debugger", func(gbc *GomeboyColor, remaining ...string) {
		gbc.debugOptions.debuggerOn = false
	})

	g.AddDebugFunc("dg", "Dump everything in graphics RAM to a file", func(gbc *GomeboyColor, remaining ...string) {
		var filename string
		if len(remaining) == 0 {
			filename = "gfxdump.png"
			//fmt.Println("No filename provided, defaulting to", filename)
		} else {
			filename = remaining[0]
		}

		f, err := os.Create(filename)

		if err != nil {
			//fmt.Println("Error creating", filename)
			//fmt.Println(err)
			return
		}
		defer f.Close()

		t0unsignedImg, _ := TilemapToImage(gbc.gpu.DumpTilemap(gpu.TILEMAP0, false), "Tilemap 0 unsigned")
		t0signedImg, _ := TilemapToImage(gbc.gpu.DumpTilemap(gpu.TILEMAP0, true), "Tilemap 0 signed")
		t1unsignedImg, _ := TilemapToImage(gbc.gpu.DumpTilemap(gpu.TILEMAP1, false), "Tilemap 1 unsigned")
		t1signedImg, _ := TilemapToImage(gbc.gpu.DumpTilemap(gpu.TILEMAP1, true), "Tilemap 1 signed")
		tilesImg, _ := TilesToImage(gbc.gpu.DumpTiles(), 512, 546)
		spritesImg, _ := SpritesToImage(gbc.gpu.Dump8x8Sprites(), 256, 546) //gbc.gpu.DumpSprites())

		out := image.NewNRGBA(image.Rect(0, 0, 1280, 546))
		draw.Draw(out, image.Rect(0, 0, 512, 546), image.NewUniform(color.White), image.ZP, draw.Src)
		draw.Draw(out, image.Rect(0, 0, 256, 273), t0unsignedImg, image.ZP, draw.Src)
		draw.Draw(out, image.Rect(0, 273, 256, 546), t1unsignedImg, image.Pt(0, 0), draw.Src)
		draw.Draw(out, image.Rect(256, 0, 512, 273), t0signedImg, image.ZP, draw.Src)
		draw.Draw(out, image.Rect(256, 273, 512, 546), t1signedImg, image.Pt(0, 0), draw.Src)
		draw.Draw(out, image.Rect(512, 0, 1024, 546), tilesImg, image.ZP, draw.Src)
		draw.Draw(out, image.Rect(1024, 0, 1280, 546), spritesImg, image.ZP, draw.Src)

		//fmt.Println("Dumping to image in file", filename)
		png.Encode(f, out)
		//fmt.Println("Done!")
	})

	g.AddDebugFunc("s", "Step", func(gbc *GomeboyColor, remaining ...string) {
		var noOfSteps int = 1
		if len(remaining) > 0 {
			val, err := strconv.ParseInt(remaining[0], 10, 64)
			if err == nil {
				noOfSteps = int(val)
			} else {
				//fmt.Println("Cannot parse argument, assuming 1 step forward instead\n\t", err)
				return
			}
		}
		//fmt.Println("Stepping forward by", noOfSteps, "instruction(s)")
		for i := 0; i < noOfSteps; i++ {
			gbc.Step()
			if g.stepDump {
				//fmt.Println(i, ":", gbc.cpu)
			}
			g.checkWatches(gbc)
		}
		//fmt.Println("Current machine state: -")
		//fmt.Println(gbc.cpu)
	})

	g.AddDebugFunc("b", "Set breakpoint", func(gbc *GomeboyColor, remaining ...string) {
		if len(remaining) == 0 {
			//fmt.Println("You must provide a PC address to break on!")
			return
		}

		var arg string = remaining[0]

		if bp, err := ToMemoryAddress(arg); err != nil {
			//fmt.Println("Could not parse memory address argument:", arg)
			//fmt.Println("\t", err)
		} else {
			//fmt.Println("Setting breakpoint to:", bp)
			g.breakWhen = bp
		}
	})

	g.AddDebugFunc("reg", "Set register", func(gbc *GomeboyColor, remaining ...string) {
		if len(remaining) < 2 {
			//fmt.Println("You must provide a register and value!")
			return
		}

		var register string = strings.ToLower(remaining[0])
		value, err := utils.StringToByte(remaining[1])

		if err != nil {
			//fmt.Println("Could not parse value", remaining[1])
			//fmt.Println("\t", err)
			return
		}

		//fmt.Println("Attempting to set register", register, "with value", utils.ByteToString(value))
		switch register {
		case "a":
			gbc.cpu.R.A = value
		case "b":
			gbc.cpu.R.B = value
		case "c":
			gbc.cpu.R.C = value
		case "d":
			gbc.cpu.R.D = value
		case "e":
			gbc.cpu.R.E = value
		case "h":
			gbc.cpu.R.H = value
		case "l":
			gbc.cpu.R.L = value
		default:
			//fmt.Println("Unknown register:", register)
		}
	})

	g.AddDebugFunc("rm", "Read data from memory", func(gbc *GomeboyColor, remaining ...string) {
		var startAddr types.Word
		switch len(remaining) {
		case 0:
			//fmt.Println("You must provide at least a starting address to inspect")
			return
		default:
			addr, err := ToMemoryAddress(remaining[0])
			if err != nil {
				//fmt.Println("Could not parse memory address: ", remaining[0])
				return
			}
			startAddr = addr
		}
		lb := startAddr - (startAddr & 0x000F)
		hb := startAddr + (0x0F - (startAddr & 0x000F))
		//fmt.Print("\t\t")
		for w := lb; w <= hb; w++ {
			//fmt.Printf("   %X ", byte(w%16))

		}
		//fmt.Println()

		//fmt.Printf("%s\t\t", lb)
		for w := lb; w <= hb; w++ {
			//fmt.Print(utils.ByteToString(gbc.mmu.ReadByte(w)), " ")
		}
		//fmt.Println()

	})

	g.AddDebugFunc("wm", "Write data to memory", func(gbc *GomeboyColor, remaining ...string) {
		var value byte
		var toAddr types.Word
		switch len(remaining) {
		case 0:
			//fmt.Println("You must provide a byte value and address to write to")
			return
		case 1:
			//fmt.Println("You must provide a byte value to put in memory")
			return
		default:
			addr, err := ToMemoryAddress(remaining[0])
			if err != nil {
				//fmt.Println("Could not parse memory address: ", remaining[0])
				return
			}
			val, err := utils.StringToByte(remaining[1])
			if err != nil {
				//fmt.Println("Could not parse value: ", remaining[1], err)
				return
			}
			toAddr = addr
			value = val
		}

		//fmt.Println("Writing", utils.ByteToString(value), "to", toAddr)
		gbc.mmu.WriteByte(toAddr, value)
	})

	g.AddDebugFunc("w", "Set memory location to watch for changes", func(gbc *GomeboyColor, remaining ...string) {
		if len(remaining) == 0 {
			//fmt.Println("You must provide a memory address to watch!")
			return
		}

		var arg string = remaining[0]

		if m, err := ToMemoryAddress(arg); err != nil {
			//fmt.Println("Could not parse memory address argument:", arg)
			//fmt.Println("\t", err)
		} else {
			//fmt.Println("Watching memory address:", m)
			value := gbc.mmu.ReadByte(m)
			g.watches[m] = value
		}
	})

	g.AddDebugFunc("q", "Quit emulator", func(gbc *GomeboyColor, remaining ...string) {
		os.Exit(0)
	})
}

func (g *DebugOptions) AddDebugFunc(command string, description string, f DebugCommandHandler) {
	g.debugFuncMap[command] = f
	g.debugHelpStr = append(g.debugHelpStr, utils.PadRight(command, 4, " ")+" = "+description)
}

func (g *DebugOptions) checkWatches(gbc *GomeboyColor) {
	for k, oldVal := range g.watches {
		currentValue := gbc.mmu.ReadByte(k)
		if oldVal != currentValue {
			//fmt.Println("Data at memory address", k, "has changed from", utils.ByteToString(oldVal), "to", utils.ByteToString(currentValue))
			//fmt.Println("Last operation:", gbc.cpu)
			g.watches[k] = currentValue
		}
	}
}

func ToMemoryAddress(s string) (types.Word, error) {
	if len(s) > 4 {
		return 0x0, errors.New("Please enter an address between 0000 and FFFF")
	}

	result, err := strconv.ParseInt(s, 16, 64)

	return types.Word(result), err
}

//IMAGE DUMP FUNCTIONS - SLOPPY CODE!!!!
func SpritesToImage(sprites [40][8][8]types.RGB, w, h int) (*image.NRGBA, error) {
	out := image.NewNRGBA(image.Rect(0, 0, w, h))

	font, err := GetFont("../resources/FreeUniversal-Regular.ttf")
	if err != nil {
		return out, err
	}
	draw.Draw(out, out.Bounds(), &image.Uniform{color.RGBA{200, 200, 200, 255}}, image.ZP, draw.Src)
	DrawTextOnImage("Sprites", font, out, 8, (out.Bounds().Dx()/2)-10, 2)
	plotX, plotY, imgX, imgY := 4, 26, 0, 0

	var spacing int = 8
	for i, spr := range sprites {
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				cr := color.RGBA{spr[y][x].Red, spr[y][x].Green, spr[y][x].Blue, 0xFF}
				out.Set(imgX+plotX, imgY+plotY, cr)
				imgX++
			}
			imgX = 0
			imgY++
		}
		imgY = 0
		DrawTextOnImage(fmt.Sprint(i), font, out, 4, plotX, plotY+12)
		plotX += 8 + spacing
		if plotX >= 490 {
			plotY += 24 + spacing
			plotX = 4
		}
	}
	return out, nil
}

func TilesToImage(tiles [512][8][8]types.RGB, w, h int) (*image.NRGBA, error) {
	out := image.NewNRGBA(image.Rect(0, 0, w, h))

	font, err := GetFont("../resources/FreeUniversal-Regular.ttf")
	if err != nil {
		return out, err
	}
	draw.Draw(out, out.Bounds(), &image.Uniform{color.RGBA{200, 200, 200, 0xFF}}, image.ZP, draw.Src)

	//draw left/right border
	for y := 0; y < h; y++ {
		out.Set(0, y, color.Black)
		out.Set(w-1, y, color.Black)
	}
	plotX, plotY, imgX, imgY := 4, 26, 0, 0
	DrawTextOnImage("Tiles", font, out, 8, (out.Bounds().Dx()/2)-10, 2)

	var spacing int = 8
	for i, tile := range tiles {
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				cr := color.RGBA{tile[y][x].Red, tile[y][x].Green, tile[y][x].Blue, 0xFF}
				out.Set(imgX+plotX, imgY+plotY, cr)
				imgX++
			}
			imgX = 0
			imgY++
		}
		imgY = 0
		DrawTextOnImage(fmt.Sprint(i), font, out, 4, plotX, plotY+12)
		plotX += 8 + spacing
		if plotX >= 490 {
			plotY += 24 + spacing
			plotX = 4
		}
	}
	return out, nil
}

func TilemapToImage(img [256][256]types.RGB, caption string) (*image.NRGBA, error) {

	out := image.NewNRGBA(image.Rect(0, 0, 256, (256 + 17)))
	draw.Draw(out, out.Bounds(), &image.Uniform{color.RGBA{235, 235, 235, 255}}, image.ZP, draw.Src)
	font, err := GetFont("../resources/FreeUniversal-Regular.ttf")
	if err != nil {
		return out, err
	}

	for i := 0; i < 256; i++ {
		for j := 0; j < 256; j++ {
			cr := img[i][j]
			out.Set(i, j, color.RGBA{cr.Red, cr.Green, cr.Blue, 0xFF})
		}
	}
	DrawTextOnImage(caption, font, out, 8, 72, 256)
	return out, nil
}

func DrawTextOnImage(text string, font *truetype.Font, img *image.NRGBA, size, x, y int) {
	c := freetype.NewContext()
	c.SetDPI(120)
	c.SetFont(font)
	c.SetFontSize(float64(size))
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.Black)
	pt := freetype.Pt(x, y+int(c.PointToFix32(float64(size))>>8))
	c.DrawString(text, pt)
}

func GetFont(location string) (*truetype.Font, error) {
	fontBytes, err := ioutil.ReadFile(location)
	if err != nil {
		return nil, err
	}
	font, err := freetype.ParseFont(fontBytes)
	if err != nil {
		return nil, err
	}
	return font, nil
}
*/
