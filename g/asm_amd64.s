#include "textflag.h"

// offset of g.m, see G in runtime2.go.
#define g_m 48

#define	get_tls(r)	MOVQ TLS, r
#define	g(r)	0(r)(TLS*1)

TEXT ·GetG(SB),NOSPLIT,$0-8
	get_tls(CX)
	MOVQ	g(CX), AX
	MOVQ	AX, ret+0(FP)
	RET

TEXT ·GetM(SB),NOSPLIT,$0-8
	get_tls(CX)
	MOVQ	g(CX), AX
	MOVQ	g_m(AX), BX
	MOVQ	BX, ret+0(FP)
	RET
