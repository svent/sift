// sift
// Copyright (C) 2014-2015 Sven Taute
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

package main

/*
#cgo CFLAGS: -std=gnu99 -O2 -funroll-loops

#include <stddef.h>

inline int count_newlines(const unsigned char *buf, size_t n) {
	int count = 0;
	int base;
	int dist = n / 4;
	for (int i = 0; i < dist; i++) {
		base = i * 4;
		if (buf[base] == 0x0a) {
			count++;
		}
		if (buf[base+1] == 0x0a) {
			count++;
		}
		if (buf[base+2] == 0x0a) {
			count++;
		}
		if (buf[base+3] == 0x0a) {
			count++;
		}
	}
	for (int i = (n / 4) * 4; i < n; i++) {
		if (buf[i] == 0x0a) {
			count += 1;
		}
	}
	return count;
}

inline void bytes_to_lower(const unsigned char *buf, unsigned char *out, size_t n) {
	int base;
	int dist = n / 4;
	for (int i = 0; i < dist; i++) {
		base = i * 4;
		out[base] = (buf[base] - 65U < 26) ? buf[base] + 32 : buf[base];
		out[base+1] = (buf[base+1] - 65U < 26) ? buf[base+1] + 32 : buf[base+1];
		out[base+2] = (buf[base+2] - 65U < 26) ? buf[base+2] + 32 : buf[base+2];
		out[base+3] = (buf[base+3] - 65U < 26) ? buf[base+3] + 32 : buf[base+3];
	}
	for (int i = (n / 4) * 4; i < n; i++) {
		out[i] = (buf[i] - 65U < 26) ? buf[i] + 32 : buf[i];
	}
}
*/
import "C"

func countNewlines(input []byte, length int) int {
	return int(C.count_newlines((*C.uchar)(&input[0]), C.size_t(length)))
}

func bytesToLower(input []byte, output []byte, length int) {
	C.bytes_to_lower((*C.uchar)(&input[0]), (*C.uchar)(&output[0]), C.size_t(length))
}
