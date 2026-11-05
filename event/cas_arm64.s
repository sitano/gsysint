#include "textflag.h"

// func cas(p *uintptr, old, new uintptr) bool
// LL/SC loop as in internal/runtime/atomic.Cas64 (no LSE requirement).
TEXT ·cas(SB),NOSPLIT,$0-25
	MOVD	p+0(FP), R0
	MOVD	old+8(FP), R1
	MOVD	new+16(FP), R2
again:
	LDAXR	(R0), R3
	CMP	R1, R3
	BNE	done
	STLXR	R2, (R0), R3
	CBNZ	R3, again
done:
	CSET	EQ, R0
	MOVB	R0, ret+24(FP)
	RET
