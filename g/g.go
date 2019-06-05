// Package g provides access to the runtime structures of go1.12.5
// that participate in organization of goroutines scheduling.
//
// On scheduling check:
// - [scheduler](https://github.com/golang/go/blob/7bc40ffb05d8813bf9b41a331b45d37216f9e747/src/runtime/proc.go#L2022)
// - [details](https://golang.org/s/go11sched)
//
// Mainly this package gives you an access to the G and P structures:
// - G - goroutine.
// - M - worker thread, or machine.
// - P - processor, a resource that is required to execute Go code.
//       M must have an associated P to execute Go code, however it can be
//       blocked or in a syscall w/o an associated P.
package g

import "unsafe"

// GetGPtr returns the pointer to the current g.
// The compiler rewrites calls to this function into instructions
// that fetch the g directly (from TLS or from the dedicated register).
func GetG() unsafe.Pointer

// GetMPtr returns the pointer to the current m.
func GetM() unsafe.Pointer

// CurG returns the pointer to the current g.
// The compiler rewrites calls to this function into instructions
// that fetch the g directly (from TLS or from the dedicated register).
func CurG() *G {
	return (*G)(GetG())
}

// CurM returns the pointer to the current m.
func CurM() *M {
	return (*M)(GetM())
}