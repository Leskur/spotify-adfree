//go:build ignore

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// 绿圆 + 透明勾（挖空）+ 4x 超采样，圆边和勾边都更顺。

func main() {
	root, _ := os.Getwd()
	outDir := filepath.Join(root, "internal", "ui", "icons")
	_ = os.MkdirAll(outDir, 0o755)

	sizes := []int{16, 32, 48, 256}
	var pngs [][]byte
	for _, s := range sizes {
		img := renderIcon(s)
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			panic(err)
		}
		pngs = append(pngs, buf.Bytes())
	}
	ico, err := encodeICO(sizes, pngs)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "tray.ico"), ico, 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote", filepath.Join(outDir, "tray.ico"))
}

func renderIcon(size int) *image.RGBA {
	const ss = 4
	hi := size * ss
	cx, cy := float64(hi)/2, float64(hi)/2
	r := float64(hi)/2 - float64(ss)*0.9

	scale := float64(hi)
	p1 := point{0.30 * scale, 0.52 * scale}
	p2 := point{0.44 * scale, 0.68 * scale}
	p3 := point{0.72 * scale, 0.36 * scale}
	thickness := 0.12 * scale
	if size <= 16 {
		thickness = 0.11 * scale
	}
	half := thickness / 2

	hiImg := image.NewRGBA(image.Rect(0, 0, hi, hi))
	green := color.RGBA{R: 0x1D, G: 0xB9, B: 0x54, A: 0xFF}

	for y := 0; y < hi; y++ {
		for x := 0; x < hi; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			if math.Hypot(fx-cx, fy-cy) > r {
				continue // 圆外透明
			}
			onCheck := distToSeg(fx, fy, p1, p2) <= half ||
				distToSeg(fx, fy, p2, p3) <= half ||
				math.Hypot(fx-p1.x, fy-p1.y) <= half ||
				math.Hypot(fx-p2.x, fy-p2.y) <= half ||
				math.Hypot(fx-p3.x, fy-p3.y) <= half
			if onCheck {
				continue // 勾挖空：透明
			}
			hiImg.SetRGBA(x, y, green)
		}
	}

	out := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var sr, sg, sb, sa uint32
			for dy := 0; dy < ss; dy++ {
				for dx := 0; dx < ss; dx++ {
					c := hiImg.RGBAAt(x*ss+dx, y*ss+dy)
					sr += uint32(c.R)
					sg += uint32(c.G)
					sb += uint32(c.B)
					sa += uint32(c.A)
				}
			}
			n := uint32(ss * ss)
			a := uint8(sa / n)
			if a == 0 {
				continue
			}
			out.SetRGBA(x, y, color.RGBA{
				R: uint8(sr / n),
				G: uint8(sg / n),
				B: uint8(sb / n),
				A: a,
			})
		}
	}
	return out
}

type point struct{ x, y float64 }

func distToSeg(px, py float64, a, b point) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	if dx == 0 && dy == 0 {
		return math.Hypot(px-a.x, py-a.y)
	}
	t := ((px-a.x)*dx + (py-a.y)*dy) / (dx*dx + dy*dy)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return math.Hypot(px-(a.x+t*dx), py-(a.y+t*dy))
}

func encodeICO(sizes []int, pngs [][]byte) ([]byte, error) {
	if len(sizes) != len(pngs) {
		return nil, fmt.Errorf("size mismatch")
	}
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(len(sizes)))
	offset := 6 + 16*len(sizes)
	for i, s := range sizes {
		w := byte(s)
		if s >= 256 {
			w = 0
		}
		buf.WriteByte(w)
		buf.WriteByte(w)
		buf.WriteByte(0)
		buf.WriteByte(0)
		_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
		_ = binary.Write(&buf, binary.LittleEndian, uint16(32))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(pngs[i])))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(offset))
		offset += len(pngs[i])
	}
	for _, p := range pngs {
		buf.Write(p)
	}
	return buf.Bytes(), nil
}
