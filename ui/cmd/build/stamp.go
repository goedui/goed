package main

import "encoding/binary"

type icoEntry struct {
	id       uint16
	w        byte
	h        byte
	colors   byte
	planes   uint16
	bitCount uint16
	data     []byte
}

func parseICO(ico []byte) []icoEntry {
	if len(ico) < 6 || binary.LittleEndian.Uint16(ico[2:4]) != 1 {
		return nil
	}
	n := int(binary.LittleEndian.Uint16(ico[4:6]))
	out := make([]icoEntry, 0, n)
	for i := 0; i < n; i++ {
		e := 6 + 16*i
		if e+16 > len(ico) {
			return nil
		}
		size := int(binary.LittleEndian.Uint32(ico[e+8:]))
		off := int(binary.LittleEndian.Uint32(ico[e+12:]))
		if size <= 0 || off < 0 || off+size > len(ico) {
			continue
		}
		out = append(out, icoEntry{
			id:       uint16(i + 1),
			w:        ico[e],
			h:        ico[e+1],
			colors:   ico[e+2],
			planes:   binary.LittleEndian.Uint16(ico[e+4:]),
			bitCount: binary.LittleEndian.Uint16(ico[e+6:]),
			data:     append([]byte(nil), ico[off:off+size]...),
		})
	}
	return out
}

func groupIcon(entries []icoEntry) []byte {
	buf := make([]byte, 6+14*len(entries))
	binary.LittleEndian.PutUint16(buf[2:], 1)
	binary.LittleEndian.PutUint16(buf[4:], uint16(len(entries)))
	for i, e := range entries {
		o := 6 + 14*i
		buf[o] = e.w
		buf[o+1] = e.h
		buf[o+2] = e.colors
		binary.LittleEndian.PutUint16(buf[o+4:], e.planes)
		binary.LittleEndian.PutUint16(buf[o+6:], e.bitCount)
		binary.LittleEndian.PutUint32(buf[o+8:], uint32(len(e.data)))
		binary.LittleEndian.PutUint16(buf[o+12:], e.id)
	}
	return buf
}
