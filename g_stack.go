package gsysint

// Information from the compiler about the layout of stack frames.
type BitVector struct {
	n        int32 // # of bits
	bytedata *uint8
}

type GoBitVector struct {
	n        uintptr
	bytedata []uint8
}