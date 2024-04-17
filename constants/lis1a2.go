package constants

type LIS1A2ConnectionStatus int

const (
	Idle         LIS1A2ConnectionStatus = iota
	Sending      LIS1A2ConnectionStatus = iota
	Receiving    LIS1A2ConnectionStatus = iota
	Establishing LIS1A2ConnectionStatus = iota
)

const (
	MaxFrameSize         = 240
	MaxConnectionRetires = 5
)
