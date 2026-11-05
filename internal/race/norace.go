//go:build !race

package race

import "unsafe"

func Acquire(unsafe.Pointer) {}
func Release(unsafe.Pointer) {}
