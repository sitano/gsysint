// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package g

import (
	"unsafe"
)

// TypeAlg is also copied/used in reflect/type.go.
// keep them in sync.
type TypeAlg struct {
	// function for hashing objects of this type
	// (ptr to object, seed) -> Hash
	Hash func(unsafe.Pointer, uintptr) uintptr
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	Equal func(unsafe.Pointer, unsafe.Pointer) bool
}

