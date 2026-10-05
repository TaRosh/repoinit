// genicon рисует иконку repoinit — «ночные врата»: красные тории
// на фоне луны и звёзды по углам.
//
// Фирменный концепт серии иконок описан в F:\style\icons\README.md.
// Неизменны: скруглённый квадрат, геометрия тории, светило по центру,
// мелкие акценты по углам, контрастная обводка ворот.
//
//	go run ./tools/genicon                      — сгенерировать иконку
//	go run ./tools/genicon -preview small.png  — ещё и превью мелких размеров ×4
//
// Пишет cmd/repoinit/icon.png (256 px, для окна) и cmd/repoinit/icon.ico (16–256 px, для exe).
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// --- Палитра: единственное, что обязательно меняется от проекта к проекту. ---

var (
	skyTop    = color.RGBA{0x12, 0x1a, 0x45, 255} // ночное небо сверху
	skyBottom = color.RGBA{0x0e, 0x5a, 0x6a, 255} // бирюза у горизонта
	moonTop   = color.RGBA{0xff, 0xf4, 0xd6, 255} // луна
	moonBot   = color.RGBA{0xf3, 0xd7, 0x8f, 255}
	starColor = color.RGBA{0xff, 0xf1, 0xc4, 255}
	toriiFill = color.RGBA{0xe2, 0x3d, 0x28, 255} // киноварь — классический цвет тории
	outline   = color.RGBA{0x12, 0x0e, 0x1a, 255} // тёмная обводка ворот
)

// --- Фирменная геометрия (НЕ менять — это узнаваемость серии). ---

type pt struct{ x, y float64 }

// Скруглённый квадрат иконки.
func inBadge(p pt) bool { return inRoundRect(p, 0.02, 0.02, 0.98, 0.98, 0.22) }

// Светило за воротами.
const sunX, sunY, sunR = 0.5, 0.5, 0.36

// inTorii — силуэт ворот тории в нормализованных координатах (0..1).
func inTorii(p pt) bool {
	return inQuad(p, [4]pt{{0.08, 0.23}, {0.92, 0.23}, {0.81, 0.33}, {0.19, 0.33}}) || // касаги — верхняя балка
		inRect(p, 0.18, 0.33, 0.82, 0.39) || // симаги — вторая балка
		inRect(p, 0.24, 0.47, 0.76, 0.53) || // нуки — нижняя перекладина
		inRect(p, 0.28, 0.39, 0.37, 0.87) || // левый столб
		inRect(p, 0.63, 0.39, 0.72, 0.87) || // правый столб
		inRect(p, 0.45, 0.39, 0.55, 0.47) // гакудзука — табличка между балками
}

// Звёзды — четырёхлучевые искры вместо лепестков сакуры: {x, y, размер}.
var stars = [][3]float64{
	{0.17, 0.13, 0.055},
	{0.84, 0.12, 0.045},
	{0.13, 0.74, 0.05},
	{0.87, 0.70, 0.055},
	{0.85, 0.90, 0.035},
}

func inStar(p pt, cx, cy, s float64) bool {
	dx, dy := math.Abs(p.x-cx), math.Abs(p.y-cy)
	// Астроида |x|^0.5 + |y|^0.5 <= s^0.5 — звезда с вогнутыми гранями.
	return math.Sqrt(dx)+math.Sqrt(dy) <= math.Sqrt(s)
}

// shade возвращает цвет точки и признак "внутри иконки".
// Слои снизу вверх: небо → луна → звёзды → (обводка+)тории.
func shade(p pt, size int) (color.RGBA, bool) {
	if !inBadge(p) {
		return color.RGBA{}, false
	}
	c := mix(skyTop, skyBottom, p.y)

	if inCircle(p, sunX, sunY, sunR) {
		c = mix(moonTop, moonBot, (p.y-(sunY-sunR))/(2*sunR))
	}
	for _, s := range stars {
		if inStar(p, s[0], s[1], s[2]) {
			c = starColor
		}
	}

	// Толщина обводки — не тоньше ~1.4 пикселя, иначе на 16 px она исчезнет.
	r := math.Max(0.026, 1.4/float64(size))
	switch {
	case inTorii(p):
		return toriiFill, true
	case near(p, r, inTorii):
		return outline, true
	}
	return c, true
}

// near — точка в пределах r от фигуры (проверка 16 точек на окружности):
// так получается обводка любой фигуры без отдельного описания её контура.
func near(p pt, r float64, in func(pt) bool) bool {
	for i := range 16 {
		a := float64(i) * math.Pi / 8
		if in(pt{p.x + r*math.Cos(a), p.y + r*math.Sin(a)}) {
			return true
		}
	}
	return false
}

// --- Примитивы. ---

func inRoundRect(p pt, x0, y0, x1, y1, r float64) bool {
	if p.x < x0 || p.x > x1 || p.y < y0 || p.y > y1 {
		return false
	}
	cx := math.Min(math.Max(p.x, x0+r), x1-r)
	cy := math.Min(math.Max(p.y, y0+r), y1-r)
	return math.Hypot(p.x-cx, p.y-cy) <= r
}

func inRect(p pt, x0, y0, x1, y1 float64) bool {
	return p.x >= x0 && p.x <= x1 && p.y >= y0 && p.y <= y1
}

func inCircle(p pt, cx, cy, r float64) bool { return math.Hypot(p.x-cx, p.y-cy) <= r }

// inQuad — точка внутри выпуклого четырёхугольника: она должна быть
// по одну сторону от всех рёбер (знак векторного произведения одинаков).
func inQuad(p pt, q [4]pt) bool {
	sign := 0.0
	for i := range 4 {
		a, b := q[i], q[(i+1)%4]
		c := (b.x-a.x)*(p.y-a.y) - (b.y-a.y)*(p.x-a.x)
		if c == 0 {
			continue
		}
		if sign == 0 {
			sign = c
		} else if sign*c < 0 {
			return false
		}
	}
	return true
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Min(math.Max(t, 0), 1)
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

// --- Растеризация и форматы. ---

// render рисует иконку size×size со сглаживанием: каждый пиксель — среднее
// по 4×4 подточкам (supersampling), поэтому края фигур не "лесенкой".
func render(size int) *image.RGBA {
	const ss = 4
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			var r, g, b, a int
			for sy := range ss {
				for sx := range ss {
					p := pt{
						(float64(x) + (float64(sx)+0.5)/ss) / float64(size),
						(float64(y) + (float64(sy)+0.5)/ss) / float64(size),
					}
					if c, ok := shade(p, size); ok {
						r += int(c.R)
						g += int(c.G)
						b += int(c.B)
						a += 255
					}
				}
			}
			if a == 0 {
				continue
			}
			img.SetRGBA(x, y, color.RGBA{
				uint8(r * 255 / a), uint8(g * 255 / a), uint8(b * 255 / a),
				uint8(a / (ss * ss)), // доля подточек внутри иконки = прозрачность края
			})
		}
	}
	return img
}

func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// encodeICO собирает .ico из PNG-кадров (формат поддерживается с Windows Vista).
// Windows сам выбирает подходящий размер: 16 — в заголовке окна, 32/48 — в
// проводнике, 256 — на большом значке.
func encodeICO(sizes []int, pngs [][]byte) []byte {
	var ico bytes.Buffer
	le := binary.LittleEndian
	_ = binary.Write(&ico, le, [3]uint16{0, 1, uint16(len(sizes))}) // заголовок: тип 1 = иконка
	offset := 6 + 16*len(sizes)
	for i, s := range sizes {
		dim := uint8(s)
		if s >= 256 {
			dim = 0 // 0 в формате ICO означает 256
		}
		ico.Write([]byte{dim, dim, 0, 0})
		_ = binary.Write(&ico, le, [2]uint16{1, 32})                                // плоскости, бит на пиксель
		_ = binary.Write(&ico, le, [2]uint32{uint32(len(pngs[i])), uint32(offset)}) // размер и смещение кадра
		offset += len(pngs[i])
	}
	for _, p := range pngs {
		ico.Write(p)
	}
	return ico.Bytes()
}

func main() {
	preview := flag.String("preview", "", "записать превью размеров 16–48 px, увеличенных в 4 раза")
	flag.Parse()

	out := filepath.Join("cmd", "repoinit")
	sizes := []int{16, 24, 32, 48, 64, 128, 256}

	pngs := make([][]byte, len(sizes))
	for i, s := range sizes {
		pngs[i] = encodePNG(render(s))
	}

	must(os.WriteFile(filepath.Join(out, "icon.png"), pngs[len(pngs)-1], 0o644))
	must(os.WriteFile(filepath.Join(out, "icon.ico"), encodeICO(sizes, pngs), 0o644))

	if *preview != "" {
		must(os.WriteFile(*preview, encodePNG(previewSheet([]int{16, 24, 32, 48}, 4)), 0o644))
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// previewSheet рисует мелкие размеры в ряд, увеличенные в zoom раз без сглаживания:
// так видно, читается ли иконка в панели задач и заголовке окна.
func previewSheet(sizes []int, zoom int) *image.RGBA {
	const gap = 10
	w := gap
	for _, s := range sizes {
		w += s*zoom + gap
	}
	maxS := sizes[len(sizes)-1] * zoom
	sheet := image.NewRGBA(image.Rect(0, 0, w, maxS+2*gap))
	draw.Draw(sheet, sheet.Bounds(), image.White, image.Point{}, draw.Src)

	x := gap
	for _, s := range sizes {
		icon := render(s)
		for y := range s * zoom {
			for xx := range s * zoom {
				sheet.Set(x+xx, gap+y, icon.At(xx/zoom, y/zoom))
			}
		}
		x += s*zoom + gap
	}
	return sheet
}
