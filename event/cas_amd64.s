#include "textflag.h"

// func cas(p *uintptr, old, new uintptr) bool
TEXT ·cas(SB),NOSPLIT,$0-25
	MOVQ	p+0(FP), BX
	MOVQ	old+8(FP), AX
	MOVQ	new+16(FP), CX
	LOCK
	CMPXCHGQ	CX, 0(BX)
	SETEQ	ret+24(FP)
	RET
