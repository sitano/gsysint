// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package g

// Information from the compiler about the layout of stack frames.
type BitVector struct {
	N        int32 // # of bits
	ByteData *uint8
}
