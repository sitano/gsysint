//go:build race

// Package race tells the race detector about happens-before edges created by
// runtime-level primitives it cannot see (runtime.lock, gopark/goready).
package race

import (
	"runtime"
	"unsafe"
)

func Acquire(p unsafe.Pointer) { runtime.RaceAcquire(p) }
func Release(p unsafe.Pointer) { runtime.RaceRelease(p) }
