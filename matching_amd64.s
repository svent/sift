// sift
// Copyright (C) 2014-2016 Sven Taute
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, version 3 of the License.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

#include "textflag.h"

// func countNewlines(input []byte, length int) int
TEXT ·countNewlines(SB),NOSPLIT,$0-40
	MOVQ input+0(FP), SI
	MOVQ length+24(FP), CX

	XORQ AX, AX
	CMPQ CX, $16
	JL small

	// X0 = 16 x 0x00
	// X1 = 16 x 0x01
	// X2 = 16 x '\n'
	// X3 = input buffer

	PXOR X0, X0

	MOVQ $0x0101010101010101, DX
	MOVQ DX, X1
	PUNPCKLQDQ X1, X1

	MOVQ $0x0a0a0a0a0a0a0a0a, DX
	MOVQ DX, X2
	PUNPCKLQDQ X2, X2

	PXOR X6, X6

	CMPQ CX, $128*16
	JL bigloop

hugeloop:
	MOVQ $128, BX
	PXOR X4, X4

hugeloop_inner:
	MOVOU (SI), X3
	PCMPEQB X2, X3
	PAND X1, X3
	PADDD X3, X4
	ADDQ $16, SI
	DECQ BX
	JNZ hugeloop_inner

	PSADBW X0, X4
	PADDQ X4, X6

	SUBQ $128*16, CX
	CMPQ CX, $128*16
	JGE hugeloop

bigloop:
	CMPQ CX, $16
	JL finish
	MOVOU (SI), X3
	PCMPEQB X2, X3
	PAND X1, X3
	PSADBW X0, X3
	PADDQ X3, X6
	SUBQ $16, CX
	ADDQ $16, SI
	JMP bigloop

finish:
	PXOR X7, X7
	MOVHLPS X6, X7
	PADDQ X6, X7
	MOVQ X7, AX

small:
	TESTQ CX, CX
	JZ done
	CMPB (SI), $'\n'
	JNE nonewline
	INCQ AX
nonewline:
	INCQ SI
	DECQ CX
	JMP small

done:
	MOVQ AX, ret+32(FP)
	RET


// func bytesToLower(input []byte, output []byte, length int)
TEXT ·bytesToLower(SB),NOSPLIT,$0-56
	MOVQ input+0(FP), SI
	MOVQ output+24(FP), DI
	MOVQ length+48(FP), CX

    // X0 = 16 x 0x00
    // X1 = 16 x 0x25 (127 - 'Z')
    // X2 = 16 x 0x65 (127 - 26)
    // X3 = 16 x 0x20 ('a' - 'A')
    // X4 = input buffer

	XORQ AX, AX
	CMPQ CX, $16
	JL small

	PXOR X0, X0

	MOVQ $0x2525252525252525, DX
	MOVQ DX, X1
	PUNPCKLQDQ X1, X1

	MOVQ $0x6565656565656565, DX
	MOVQ DX, X2
	PUNPCKLQDQ X2, X2

	MOVQ $0x2020202020202020, DX
	MOVQ DX, X3
	PUNPCKLQDQ X3, X3

	XORQ BX, BX
bigloop:
	MOVOU (SI)(BX*8), X4
	MOVOU X4, X5
	PADDB X1, X5
	PCMPGTB X2, X5
	PAND X3, X5
	PADDB X5, X4
	MOVOU X4, (DI)(BX*8)

	ADDQ $2, BX
	SUBQ $16, CX
	CMPQ CX, $16
	JGE bigloop

	TESTQ CX, CX
	JZ done

	SUBQ $16, CX
	ADDQ CX, SI
	ADDQ CX, DI
	MOVOU (SI)(BX*8), X4
	MOVOU X4, X5
	PADDB X1, X5
	PCMPGTB X2, X5
	PAND X3, X5
	PADDB X5, X4
	MOVOU X4, (DI)(BX*8)
	JMP done

small:
	CLD
loop_small:
	TESTQ CX, CX
	JZ done
	MOVB (SI), AX
	CMPB AX, $'Z'
	JA skip
	CMPB AX, $'A'
	JB skip
	ADDB $0x20, AX
	MOVB AX, (DI)
	INCQ SI
	INCQ DI
	DECQ CX
	JMP loop_small
skip:
	MOVSB
	DECQ CX
	JMP loop_small

done:
	RET


