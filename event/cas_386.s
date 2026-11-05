#include "textflag.h"

// func cas(p *uintptr, old, new uintptr) bool
TEXT ·cas(SB),NOSPLIT,$0-13
	MOVL	p+0(FP), BX
	MOVL	old+4(FP), AX
	MOVL	new+8(FP), CX
	LOCK
	CMPXCHGL	CX, 0(BX)
	SETEQ	ret+12(FP)
	RET
