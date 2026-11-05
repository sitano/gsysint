#include "textflag.h"

// offset of g.m, see G in runtime2.go.
#define g_m 24

#define	get_tls(r)	MOVL TLS, r
#define	g(r)	0(r)(TLS*1)

TEXT ·GetG(SB),NOSPLIT,$0-4
	get_tls(CX)
	MOVL	g(CX), AX
	MOVL	AX, ret+0(FP)
	RET

TEXT ·GetM(SB),NOSPLIT,$0-4
	get_tls(CX)
	MOVL	g(CX), AX
	MOVL	g_m(AX), BX
	MOVL	BX, ret+0(FP)
	RET
