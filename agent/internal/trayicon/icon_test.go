package trayicon

import (
	"encoding/binary"
	"testing"
)

func TestICOHeaderIsValid(t *testing.T) {
	ico := ICO()

	if binary.LittleEndian.Uint16(ico[2:]) != 1 {
		t.Fatal("type must be 1 (icon)")
	}
	if binary.LittleEndian.Uint16(ico[4:]) != 1 {
		t.Fatal("expected exactly one image")
	}
	if ico[6] != size || ico[7] != size {
		t.Fatalf("entry size = %dx%d", ico[6], ico[7])
	}
	bytesInRes := binary.LittleEndian.Uint32(ico[14:])
	offset := binary.LittleEndian.Uint32(ico[18:])
	if int(offset+bytesInRes) != len(ico) {
		t.Fatalf("offset %d + size %d != len %d", offset, bytesInRes, len(ico))
	}
	if binary.LittleEndian.Uint32(ico[offset:]) != 40 {
		t.Fatal("image must start with a BITMAPINFOHEADER")
	}
}

func TestCornersAreTransparentCentreIsOpaque(t *testing.T) {
	px := pixels()
	alpha := func(x, y int) byte { return px[(y*size+x)*4+3] }

	if alpha(0, 0) != 0 {
		t.Fatal("corner should be transparent")
	}
	if alpha(size/2, size/2) != 255 {
		t.Fatal("centre should be opaque")
	}
}
