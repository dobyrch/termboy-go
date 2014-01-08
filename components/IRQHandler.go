package components

type IRQHandler interface {
	RequestInterrupt(interrupt byte)
}
