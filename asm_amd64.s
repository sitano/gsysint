#include "textflag.h"
#include "go_asm_amd64.h"

#define	get_tls(r)	MOVQ TLS, r
#define	g(r)	0(r)(TLS*1)

TEXT ·getg(SB),NOSPLIT,$0-8
	get_tls(CX)
	MOVQ	g(CX), AX
	MOVQ	AX, gp+0(FP)
	RET

TEXT ·getm(SB),NOSPLIT,$0-8
	get_tls(CX)
	MOVQ	g(CX), AX
	MOVQ	g_m-8(AX), BX
	MOVQ	BX, mp+0(FP)
	RET
