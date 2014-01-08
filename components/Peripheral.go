package components

import "github.com/dobyrch/termboy-go/types"

type Peripheral interface {
	Name() string
	Read(Address types.Word) byte
	Write(Address types.Word, Value byte)
	LinkIRQHandler(m IRQHandler)
	Reset()
}
