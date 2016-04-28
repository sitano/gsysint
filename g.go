package gsysint

import "unsafe"

// getg returns the pointer to the current g.
// The compiler rewrites calls to this function into instructions
// that fetch the g directly (from TLS or from the dedicated register).
func getg() (gp unsafe.Pointer)