package classmedia

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
)

// makeThumbnail returns a JPEG no wider or taller than max pixels, turned
// upright using the photo's EXIF orientation (phones store most photos
// sideways plus a rotation tag). Formats the standard library can't decode
// (WEBP, HEIC) return an error; the app then shows the original.
func makeThumbnail(data []byte, max int) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return nil, errors.New("empty image")
	}
	scale := 1.0
	if w > max || h > max {
		if w > h {
			scale = float64(max) / float64(w)
		} else {
			scale = float64(max) / float64(h)
		}
	}
	tw, th := maxInt(1, int(float64(w)*scale)), maxInt(1, int(float64(h)*scale))
	small := downscale(src, tw, th)
	upright := orient(small, exifOrientation(data))
	var out bytes.Buffer
	if err := jpeg.Encode(&out, upright, &jpeg.Options{Quality: 78}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// downscale averages each target pixel's source box (a box filter), which
// looks clean for the large reductions thumbnails need.
func downscale(src image.Image, tw, th int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	for ty := 0; ty < th; ty++ {
		y0, y1 := b.Min.Y+ty*h/th, b.Min.Y+(ty+1)*h/th
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for tx := 0; tx < tw; tx++ {
			x0, x1 := b.Min.X+tx*w/tw, b.Min.X+(tx+1)*w/tw
			if x1 <= x0 {
				x1 = x0 + 1
			}
			// Sample at most 4x4 points per box: plenty for a thumbnail and
			// keeps large photos fast.
			var r, g, bl, a, n uint64
			sy := maxInt(1, (y1-y0)/4)
			sx := maxInt(1, (x1-x0)/4)
			for y := y0; y < y1; y += sy {
				for x := x0; x < x1; x += sx {
					cr, cg, cb, ca := src.At(x, y).RGBA()
					r, g, bl, a, n = r+uint64(cr), g+uint64(cg), bl+uint64(cb), a+uint64(ca), n+1
				}
			}
			dst.Set(tx, ty, color.RGBA64{uint16(r / n), uint16(g / n), uint16(bl / n), uint16(a / n)})
		}
	}
	return dst
}

// orient applies an EXIF orientation (1-8) to img.
func orient(img *image.RGBA, o int) image.Image {
	if o < 2 || o > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	swap := o >= 5
	dw, dh := w, h
	if swap {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2: // mirror horizontal
				dx, dy = w-1-x, y
			case 3: // rotate 180
				dx, dy = w-1-x, h-1-y
			case 4: // mirror vertical
				dx, dy = x, h-1-y
			case 5: // transpose
				dx, dy = y, x
			case 6: // rotate 90 clockwise
				dx, dy = h-1-y, x
			case 7: // transverse
				dx, dy = h-1-y, w-1-x
			case 8: // rotate 90 counter-clockwise
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, img.At(x, y))
		}
	}
	return dst
}

// exifOrientation reads the Orientation tag (0x0112) from a JPEG's APP1
// Exif segment; 1 (upright) when there is none.
func exifOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if marker == 0xDA || size < 2 || i+2+size > len(data) {
			return 1 // start of image data, or a broken segment
		}
		seg := data[i+4 : i+2+size]
		if marker == 0xE1 && len(seg) > 14 && string(seg[:6]) == "Exif\x00\x00" {
			return tiffOrientation(seg[6:])
		}
		i += 2 + size
	}
	return 1
}

func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	ifd := int(bo.Uint32(t[4:8]))
	if ifd+2 > len(t) {
		return 1
	}
	n := int(bo.Uint16(t[ifd : ifd+2]))
	for k := 0; k < n; k++ {
		e := ifd + 2 + k*12
		if e+12 > len(t) {
			return 1
		}
		if bo.Uint16(t[e:e+2]) == 0x0112 {
			v := int(bo.Uint16(t[e+8 : e+10]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
