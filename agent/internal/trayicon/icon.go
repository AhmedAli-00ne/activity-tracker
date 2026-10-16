// Package trayicon draws the tray icon in code so the agent ships as one file.
package trayicon

import (
	"bytes"
	"encoding/binary"
)

const size = 32

// Ring colour (BGR byte order is applied when writing pixels).
var ring = [3]byte{0x2E, 0x9E, 0x6B} // r, g, b

// pixels returns top-down RGBA pixels: a solid dot inside a ring, which reads
// as a "recording" indicator at tray size.
func pixels() []byte {
	px := make([]byte, size*size*4)
	c := float64(size-1) / 2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := float64(x)-c, float64(y)-c
			d2 := dx*dx + dy*dy
			inRing := d2 <= 15*15 && d2 >= 11*11
			inDot := d2 <= 7*7
			if !inRing && !inDot {
				continue
			}
			i := (y*size + x) * 4
			px[i], px[i+1], px[i+2], px[i+3] = ring[0], ring[1], ring[2], 255
		}
	}
	return px
}

// ICO returns a single 32x32 32-bit icon in .ico format.
func ICO() []byte {
	px := pixels()
	maskRow := ((size + 31) / 32) * 4
	imgLen := 40 + len(px) + maskRow*size

	var b bytes.Buffer
	w := func(v any) { _ = binary.Write(&b, binary.LittleEndian, v) }

	// ICONDIR
	w(uint16(0))
	w(uint16(1))
	w(uint16(1))
	// ICONDIRENTRY
	w([4]byte{size, size, 0, 0})
	w(uint16(1))
	w(uint16(32))
	w(uint32(imgLen))
	w(uint32(6 + 16))
	// BITMAPINFOHEADER (height is doubled to cover the AND mask)
	w(uint32(40))
	w(int32(size))
	w(int32(size * 2))
	w(uint16(1))
	w(uint16(32))
	w(uint32(0))
	w(uint32(len(px) + maskRow*size))
	w([16]byte{})

	// Pixels bottom-up, BGRA.
	for y := size - 1; y >= 0; y-- {
		for x := 0; x < size; x++ {
			i := (y*size + x) * 4
			b.Write([]byte{px[i+2], px[i+1], px[i], px[i+3]})
		}
	}
	// AND mask: all zero, alpha channel does the work.
	b.Write(make([]byte, maskRow*size))
	return b.Bytes()
}
