package photo

// HEIC/HEIF metadata extraction (phase 11). HEIF is an ISO Base Media File
// Format container: a sequence of length-prefixed boxes, with metadata items
// declared in the `meta` box — `iinf` names each item (the one typed "Exif"
// carries the EXIF payload), `iloc` says where its bytes live in the file.
// Extracting EXIF is therefore a bounds-checked box walk ending in the same
// TIFF parser JPEG uses; the pixels are HEVC-encoded and never touched —
// ingestion needs positions and instants, not pixels (BRIEF §1.3), which is
// why this file contains no codec.
//
// The same posture as exif.go: malformation is absence, never a crash. Files
// in the wild truncate boxes, point extents past EOF, and use construction
// methods this walk does not speak; every such case returns an empty Meta.

import "encoding/binary"

// ExtractHEIF walks a HEIF container and returns whatever its Exif item
// holds of the consumed tags. The caller has established via Sniff that data
// is a HEIF-family container.
func ExtractHEIF(data []byte) Meta {
	var m Meta
	tiff, ok := findHEIFExifTIFF(data)
	if !ok {
		return m
	}
	parseTIFF(tiff, &m)
	return m
}

// findHEIFExifTIFF locates the Exif item's TIFF blob: top-level boxes to
// `meta`, its `iinf` for the Exif item's ID, its `iloc` for the extent, then
// the ExifDataBlock at that file offset (a 4-byte offset field, then the
// payload the offset points into).
func findHEIFExifTIFF(data []byte) ([]byte, bool) {
	meta, ok := findBox(data, 0, len(data), "meta")
	if !ok || len(meta) < 4 {
		return nil, false
	}
	meta = meta[4:] // meta is a FullBox: skip version/flags

	iinf, ok := findBox(meta, 0, len(meta), "iinf")
	if !ok {
		return nil, false
	}
	exifID, ok := exifItemID(iinf)
	if !ok {
		return nil, false
	}
	iloc, ok := findBox(meta, 0, len(meta), "iloc")
	if !ok {
		return nil, false
	}
	off, length, ok := itemExtent(iloc, exifID)
	if !ok || off+length > len(data) || length < 4 {
		return nil, false
	}
	item := data[off : off+length]

	// ExifDataBlock: unsigned int(32) exif_tiff_header_offset, then the
	// payload; the TIFF header begins that many bytes into the payload. Some
	// writers wrap the classic "Exif\0\0" prefix inside, some do not —
	// accept both, verified by the TIFF magic itself.
	prefix := int(binary.BigEndian.Uint32(item[:4]))
	for _, start := range []int{4 + prefix, 4 + prefix + 6} {
		if start >= 4 && start+4 <= len(item) && isTIFFMagic(item[start:]) {
			return item[start:], true
		}
	}
	if len(item) > 10 && string(item[4:10]) == "Exif\x00\x00" && isTIFFMagic(item[10:]) {
		return item[10:], true
	}
	return nil, false
}

func isTIFFMagic(b []byte) bool {
	return len(b) >= 4 && (string(b[:4]) == "II*\x00" || string(b[:4]) == "MM\x00*")
}

// findBox scans the box sequence in data[start:end] for the first box of the
// given type and returns its payload. Size 0 (to end of enclosing space) and
// size 1 (64-bit largesize) are handled; anything inconsistent stops the
// scan — absence, not error.
func findBox(data []byte, start, end int, typ string) ([]byte, bool) {
	i := start
	for i+8 <= end && end <= len(data) {
		size := int(binary.BigEndian.Uint32(data[i : i+4]))
		boxType := string(data[i+4 : i+8])
		payloadStart := i + 8
		var boxEnd int
		switch size {
		case 0:
			boxEnd = end
		case 1:
			if i+16 > end {
				return nil, false
			}
			size64 := binary.BigEndian.Uint64(data[i+8 : i+16])
			if size64 > uint64(end-i) {
				return nil, false
			}
			payloadStart = i + 16
			boxEnd = i + int(size64)
		default:
			if size < 8 || i+size > end {
				return nil, false
			}
			boxEnd = i + size
		}
		if boxType == typ {
			return data[payloadStart:boxEnd], true
		}
		i = boxEnd
	}
	return nil, false
}

// exifItemID reads an iinf payload and returns the item_ID of the first
// entry whose item_type is "Exif".
func exifItemID(iinf []byte) (int, bool) {
	if len(iinf) < 6 {
		return 0, false
	}
	version := iinf[0]
	i := 4
	var count int
	if version == 0 {
		count = int(binary.BigEndian.Uint16(iinf[i : i+2]))
		i += 2
	} else {
		if len(iinf) < 8 {
			return 0, false
		}
		count = int(binary.BigEndian.Uint32(iinf[i : i+4]))
		i += 4
	}
	for range count {
		if i+8 > len(iinf) {
			return 0, false
		}
		size := int(binary.BigEndian.Uint32(iinf[i : i+4]))
		if size < 8 || i+size > len(iinf) || string(iinf[i+4:i+8]) != "infe" {
			return 0, false
		}
		entry := iinf[i+8 : i+size]
		if len(entry) < 4 {
			return 0, false
		}
		ver := entry[0]
		j := 4
		var id int
		switch ver {
		case 2:
			if j+8 > len(entry) {
				return 0, false
			}
			id = int(binary.BigEndian.Uint16(entry[j : j+2]))
			j += 4 // item_ID + item_protection_index
		case 3:
			if j+10 > len(entry) {
				return 0, false
			}
			id = int(binary.BigEndian.Uint32(entry[j : j+4]))
			j += 6
		default:
			// v0/v1 entries carry no item_type in the form this walk reads.
			i += size
			continue
		}
		if j+4 <= len(entry) && string(entry[j:j+4]) == "Exif" {
			return id, true
		}
		i += size
	}
	return 0, false
}

// itemExtent reads an iloc payload and returns the absolute file offset and
// length of the named item's first extent. Only construction method 0 (file
// offset) is spoken — cameras write that; anything else is absence.
func itemExtent(iloc []byte, itemID int) (off, length int, ok bool) {
	if len(iloc) < 8 {
		return 0, 0, false
	}
	version := iloc[0]
	offSize := int(iloc[4] >> 4)
	lenSize := int(iloc[4] & 0xF)
	baseSize := int(iloc[5] >> 4)
	indexSize := 0
	if version == 1 || version == 2 {
		indexSize = int(iloc[5] & 0xF)
	}
	i := 6
	var count int
	if version < 2 {
		count = int(binary.BigEndian.Uint16(iloc[i : i+2]))
		i += 2
	} else {
		if i+4 > len(iloc) {
			return 0, 0, false
		}
		count = int(binary.BigEndian.Uint32(iloc[i : i+4]))
		i += 4
	}
	readN := func(n int) (int, bool) {
		if n == 0 {
			return 0, true
		}
		if i+n > len(iloc) || n > 8 {
			return 0, false
		}
		v := 0
		for _, b := range iloc[i : i+n] {
			v = v<<8 | int(b)
		}
		i += n
		return v, true
	}
	for range count {
		var id int
		if version < 2 {
			v, ok := readN(2)
			if !ok {
				return 0, 0, false
			}
			id = v
		} else {
			v, ok := readN(4)
			if !ok {
				return 0, 0, false
			}
			id = v
		}
		method := 0
		if version == 1 || version == 2 {
			v, ok := readN(2)
			if !ok {
				return 0, 0, false
			}
			method = v & 0xF
		}
		if _, ok := readN(2); !ok { // data_reference_index
			return 0, 0, false
		}
		base, ok := readN(baseSize)
		if !ok {
			return 0, 0, false
		}
		extents, ok := readN(2)
		if !ok {
			return 0, 0, false
		}
		for e := range extents {
			if _, ok := readN(indexSize); !ok {
				return 0, 0, false
			}
			eo, ok := readN(offSize)
			if !ok {
				return 0, 0, false
			}
			el, ok := readN(lenSize)
			if !ok {
				return 0, 0, false
			}
			if id == itemID && method == 0 && e == 0 {
				return base + eo, el, true
			}
		}
		if id == itemID {
			return 0, 0, false // found, but not in a form this walk speaks
		}
	}
	return 0, 0, false
}
