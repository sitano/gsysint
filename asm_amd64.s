#include "textflag.h"

#define	get_tls(r)	MOVQ TLS, r
#define	g(r)	0(r)(TLS*1)

TEXT ·getg(SB),NOSPLIT,$0-8
	get_tls(CX)
	MOVQ	g(CX), AX
	MOVQ	AX, gp+0(FP)
	RET
