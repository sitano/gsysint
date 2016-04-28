#include "textflag.h"
#include "go_asm_386.h"

#define	get_tls(r)	MOVL TLS, r
#define	g(r)	0(r)(TLS*1)

TEXT ·getg(SB),NOSPLIT,$0-4
	get_tls(CX)
	MOVL	g(CX), AX
	MOVL	AX, gp+0(FP)
	RET

TEXT ·getm(SB),NOSPLIT,$0-4
	get_tls(CX)
	MOVL	g(CX), AX
	MOVL	g_m-4(AX), BX
	MOVL	BX, gp+0(FP)
	RET