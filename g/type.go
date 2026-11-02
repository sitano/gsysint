// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Runtime type representation.

package g

import "unsafe"

// tflag is documented in reflect/type.go.
// Mirrors internal/abi.TFlag.
type tflag uint8

const (
	tflagUncommon       tflag = 1 << 0
	tflagExtraStar      tflag = 1 << 1
	tflagNamed          tflag = 1 << 2
	tflagRegularMemory  tflag = 1 << 3
	tflagGCMaskOnDemand tflag = 1 << 4
	tflagDirectIface    tflag = 1 << 5
)

// Type mirrors internal/abi.Type (runtime._type).
type Type struct {
	size       uintptr
	ptrdata    uintptr // size of memory prefix holding all pointers
	hash       uint32
	tflag      tflag
	align      uint8
	fieldalign uint8
	kind       uint8
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	equal func(unsafe.Pointer, unsafe.Pointer) bool
	// gcdata stores the GC type data for the garbage collector.
	gcdata    *byte
	str       nameOff
	ptrToThis typeOff
}

type nameOff int32
type typeOff int32
type textOff int32
