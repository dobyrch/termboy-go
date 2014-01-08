package timer

import (
	"github.com/dobyrch/termboy-go/components"
	"github.com/dobyrch/termboy-go/constants"
	"fmt"
	"log"
	"github.com/dobyrch/termboy-go/types"
)

const (
	DIV_REGISTER  types.Word = 0xFF04
	TIMA_REGISTER            = 0xFF05
	TMA_REGISTER             = 0xFF06
	TAC_REGISTER             = 0xFF07
)

const (
	NAME = "TIMER"
)

type Frequency string

const (
	freq4096   Frequency = "4096hz"
	freq16384            = "16384hz"
	freq65536            = "65536hz"
	freq262144           = "262144hz"
)

var FrequenciesToCycles map[Frequency]int = map[Frequency]int{
	freq4096:   256, //1024,
	freq262144: 4,
	freq65536:  16,
	freq16384:  64,
}

type Counter struct {
	Name           string
	ClockFrequency Frequency
	ClockCounter   int
	Value          byte
}

func NewCounter(name string, initialFreq Frequency) *Counter {
	var counter *Counter = new(Counter)
	counter.Reset()
	counter.Name = name
	counter.SetFrequency(initialFreq)
	return counter
}

func (c *Counter) String() string {
	return fmt.Sprint(c.Name+":", " Frequency: ", c.ClockFrequency, " (", FrequenciesToCycles[c.ClockFrequency], " cycles) ", "| Current Counter: ", c.ClockCounter, " | Value: ", c.Value)
}

func (c *Counter) Reset() {
	//default to 4096
	c.SetFrequency(freq4096)
	c.Value = 0
}

func (c *Counter) SetFrequency(freq Frequency) {
	c.ClockFrequency = freq
	c.ClockCounter = FrequenciesToCycles[freq]
}

//returns true when overflowed
func (c *Counter) Step(cycles int) bool {
	c.ClockCounter -= cycles

	for c.ClockCounter <= 0 {
		c.Value++
		c.ClockCounter += FrequenciesToCycles[c.ClockFrequency]
		if c.Value == 0x00 {
			return true
		}
	}

	return false
}

type Timer struct {
	timaCounter *Counter
	divCounter  *Counter

	tacRegister     byte
	tmaRegister     byte
	irqHandler      components.IRQHandler
	interruptThrown bool
}

func NewTimer() *Timer {
	var t *Timer = new(Timer)
	t.divCounter = NewCounter("DIV", freq16384)
	t.timaCounter = NewCounter("TIMA", freq4096)
	t.tacRegister = 0x00
	t.tmaRegister = 0x00
	return t
}

func (timer *Timer) Name() string {
	return NAME
}

func (timer *Timer) Step(cycles int) {
	if timer.tacRegister&0x04 == 0x04 {
		if overflowed := timer.timaCounter.Step(cycles); overflowed {
			timer.irqHandler.RequestInterrupt(constants.TIMER_OVERFLOW_IRQ)
			timer.timaCounter.Value = timer.tmaRegister
		}
	}

	//step div timer
	timer.divCounter.Step(cycles)
}

func (timer *Timer) Read(Address types.Word) byte {
	switch Address {
	case DIV_REGISTER:
		return timer.divCounter.Value
	case TIMA_REGISTER:
		return timer.timaCounter.Value
	case TMA_REGISTER:
		return timer.tmaRegister
	case TAC_REGISTER:
		return timer.tacRegister
	default:
		panic(fmt.Sprintln("Timer module is not set up to handle address", Address))
	}
}

func (timer *Timer) Write(address types.Word, value byte) {
	switch address {
	case DIV_REGISTER:
		timer.divCounter.Value = 0
	case TIMA_REGISTER:
		timer.timaCounter.Value = value
	case TMA_REGISTER:
		timer.tmaRegister = value
	case TAC_REGISTER:
		var oldFrequency Frequency = timer.GetFrequency(timer.tacRegister & 0x03)
		var newFrequency Frequency = timer.GetFrequency(value & 0x03)

		if oldFrequency != newFrequency {
			timer.timaCounter.SetFrequency(newFrequency)
			log.Println(timer.timaCounter)
		}

		timer.tacRegister = value
	default:
		panic(fmt.Sprintln("Timer module is not set up to handle address", address))
	}
}

func (timer *Timer) GetFrequency(freqId byte) Frequency {
	switch freqId {
	case 0:
		return freq4096
	case 1:
		return freq262144
	case 2:
		return freq65536
	case 3:
		return freq16384
	default:
		panic("Unknown frequency!")
	}
}

func (timer *Timer) LinkIRQHandler(m components.IRQHandler) {
	timer.irqHandler = m
	log.Println(timer.Name() + ": Linked IRQ Handler to Timer")
}

func (timer *Timer) Reset() {
	log.Println("Resetting", timer.Name())
}
