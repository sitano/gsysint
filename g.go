package gsysint

import "unsafe"

// GetG returns the pointer to the current g.
// The compiler rewrites calls to this function into instructions
// that fetch the g directly (from TLS or from the dedicated register).
func GetG() (gp unsafe.Pointer)

// GetM returns the pointer to the current m.
func GetM() (mp unsafe.Pointer)