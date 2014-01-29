package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/dobyrch/termboy-go/utils"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	SKIP_BOOT_FLAG   string = "skipboot"
	SCREEN_SIZE_FLAG        = "size"
	SHOW_FPS_FLAG           = "showfps"
	TITLE_FLAG              = "title"
	DUMP_FLAG               = "dump"
	DEBUGGER_ON_FLAG        = "debug"
	BREAK_WHEN_FLAG         = "b"
	COLOR_MODE_FLAG         = "color"
	HELP_FLAG               = "help"
)

var title *string = flag.String(TITLE_FLAG, TITLE, "Title to use")
var fps *bool = flag.Bool(SHOW_FPS_FLAG, false, "Calculate and display frames per second")
var screenSizeMultiplier *int = flag.Int(SCREEN_SIZE_FLAG, 1, "Screen size multiplier")
var skipBoot *bool = flag.Bool(SKIP_BOOT_FLAG, false, "Skip boot sequence")
var colorMode *bool = flag.Bool(COLOR_MODE_FLAG, false, "Emulates Gameboy Color Hardware")
var help *bool = flag.Bool(HELP_FLAG, false, "Show this help message")

//debug stuff...
var dumpState *bool = flag.Bool(DUMP_FLAG, false, "Print state of machine after each cycle (WARNING - WILL RUN SLOW)")
var debug *bool = flag.Bool(DEBUGGER_ON_FLAG, false, "Enable debugger")
var breakOn *string = flag.String(BREAK_WHEN_FLAG, "0x0000", "Break into debugger when PC equals a given value between 0x0000 and 0xFFFF")

func PrintHelp() {
	log.Println("\nUsage: -\n")
	log.Println("To launch the emulator, simply run and pass it the location of your ROM file, e.g. ")
	log.Println("\n\tgomeboycolor location/of/romfile.gbc\n")
	log.Println("Flags: -\n")
	log.Println("	-help			->	Show this help message")
	log.Println("	-skipboot		->	Disables the boot sequence and will boot you straight into the ROM you have provided. Defaults to false")
	log.Println("	-color			->	Turns color GB features on. Defaults to true")
	log.Println("	-showfps		->	Prints average frames per second to the console. Defaults to false")
	log.Println("	-dump			-> 	Dump CPU state after every cycle. Will be very SLOW and resource intensive. Defaults to false")
	log.Println("	-size=(1-6)		->	Set screen size. Defaults to 1.")
	log.Println("	-title=(title)		->	Change window title. Defaults to 'gomeboycolor'.")
	log.Println("\nYou can pass an option argument to the boolean flags if you want to enable that particular option. e.g. to disable the boot screen you would do the following")
	log.Println("\n\tgomeboycolor -skipboot=false location/of/romfile.gbc\n")
}

type Config struct {
	//mandatory settings
	Title      string
	ScreenSize int
	SkipBoot   bool
	DisplayFPS bool
	ColorMode  bool

	//optional
	Debug     bool
	BreakOn   string
	DumpState bool

	//internal
	SettingsDir string
	SavesDir    string
}

func NewConfig() *Config {
	var c *Config = new(Config)
	return c
}

func (c *Config) LoadConfig() error {
	settingsFilepath := filepath.Join(c.SettingsDir, "config.json")
	if ok, _ := utils.Exists(settingsFilepath); ok {
		log.Println("Loading configuration from file:", settingsFilepath)

		file, err := ioutil.ReadFile(settingsFilepath)
		if err != nil {
			return err
		}

		//Make sure all settings keys are present
		var initialMap map[string]interface{}
		err = json.Unmarshal(file, &initialMap)

		if err != nil {
			return ConfigSettingsParseError(fmt.Sprintf("%v", err))
		}

		for _, v := range []string{"Title", "ScreenSize", "ColorMode", "SkipBoot", "DisplayFPS"} {
			if _, ok := initialMap[v]; !ok {
				return ConfigValidationError("Could not find settings key: \"" + v + "\" in settings file")
			}
		}

		//Now parse into struct
		err = json.Unmarshal(file, &c)

		if err != nil {
			return err
		}

		//Perform validations
		err = c.Validate()
		if err != nil {
			return err
		}

		//these are defaults
		c.Debug = *debug
		c.BreakOn = *breakOn
		c.DumpState = *dumpState
	} else {
		log.Println("Could not find settings file at", settingsFilepath, "using default values instead...")
		c.LoadDefaultConfig()
	}
	return nil
}

func (c *Config) LoadDefaultConfig() {
	c.ScreenSize = *screenSizeMultiplier
	c.SkipBoot = *skipBoot
	c.DisplayFPS = *fps
	c.Title = *title
	c.Debug = *debug
	c.BreakOn = *breakOn
	c.DumpState = *dumpState
	c.ColorMode = *colorMode
}

func (c *Config) String() string {
	return fmt.Sprintln("Configuration settings") +
		fmt.Sprintln(strings.Repeat("-", 50)) +
		fmt.Sprintln(utils.PadRight("Title: ", 19, " "), c.Title) +
		fmt.Sprintln(utils.PadRight("Skip Boot: ", 19, " "), c.SkipBoot) +
		fmt.Sprintln(utils.PadRight("GB Color Mode: ", 19, " "), c.ColorMode) +
		fmt.Sprintln(utils.PadRight("Display FPS: ", 19, " "), c.DisplayFPS) +
		fmt.Sprintln(utils.PadRight("Screen Size: ", 19, " "), c.ScreenSize) +
		fmt.Sprintln(utils.PadRight("Debug mode?: ", 19, " "), c.Debug) +
		fmt.Sprintln(utils.PadRight("Breakpoint: ", 19, " "), c.BreakOn) +
		fmt.Sprintln(utils.PadRight("CPU Dump?: ", 19, " "), c.DumpState) +
		fmt.Sprintln(utils.PadRight("Settings Dir: ", 19, " "), c.SettingsDir) +
		fmt.Sprintln(utils.PadRight("Saves Dir: ", 19, " "), c.SavesDir) +
		fmt.Sprint(strings.Repeat("-", 50))
}

func (c *Config) Validate() error {
	if c.Title == "" {
		return ConfigValidationError("\"Title\" attribute cannot be blank")
	}

	if c.ScreenSize <= 0 || c.ScreenSize > 6 {
		return ConfigValidationError("\"ScreenSize\" attribute must be between 1 and 6")
	}

	return nil
}

func (currentConfig *Config) OverrideConfigWithAnySetFlags() {
	overrideFn := func(f *flag.Flag) {
		log.Println("Overriding configuration in settings file for flag: -" + f.Name)
		switch f.Name {
		case SKIP_BOOT_FLAG:
			currentConfig.SkipBoot = *skipBoot
		case SCREEN_SIZE_FLAG:
			currentConfig.ScreenSize = *screenSizeMultiplier
		case SHOW_FPS_FLAG:
			currentConfig.DisplayFPS = *fps
		case TITLE_FLAG:
			currentConfig.Title = *title
		case COLOR_MODE_FLAG:
			currentConfig.ColorMode = *colorMode
		}
	}
	flag.Visit(overrideFn)
}

func ConfigValidationError(message string) error {
	return errors.New(fmt.Sprintf("Config validation error: %s", message))
}

func ConfigSettingsParseError(message string) error {
	return errors.New(fmt.Sprintf("Config parse error: %s", message))
}

func (config *Config) ConfigureSettingsDirectory() error {
	usr, err := user.Current()

	if err != nil {
		return err
	}

	config.SettingsDir = filepath.Join(usr.HomeDir, ".config/"+TITLE)

	if ok, _ := utils.Exists(config.SettingsDir); !ok {
		log.Println("Creating settings directory: ", config.SettingsDir)
		if err := os.Mkdir(config.SettingsDir, 0755); err != nil {
			return err
		}
	}

	config.SavesDir = filepath.Join(config.SettingsDir, "saves")

	if ok, _ := utils.Exists(config.SavesDir); !ok {
		log.Println("Creating saves directory: ", config.SavesDir)
		if err := os.Mkdir(config.SavesDir, 0755); err != nil {
			return err
		}
	}

	logPath := filepath.Join(config.SettingsDir, "log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}

	log.SetOutput(logFile)

	return nil
}
