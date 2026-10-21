// gifpoc renders the CORTEX #493 review: two vector hands, no effect layers.
// It deliberately writes only review.gif. All artwork and the stroke font are
// embedded; the renderer uses only the standard library.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"math"
	"os"
	"path/filepath"
	"sort"
)

const (
	width   = 720
	height  = 405
	samples = 3
)

type point struct{ x, y float64 }
type path []point

func move(x, y float64) path          { return path{{x, y}} }
func (p path) line(x, y float64) path { return append(p, point{x, y}) }

// cubic flattens a Bezier at subpixel resolution before supersampled rasterizing.
func (p path) cubic(a, b, c, d, e, f float64) path {
	start := p[len(p)-1]
	for i := 1; i <= 24; i++ {
		t := float64(i) / 24
		u := 1 - t
		p = append(p, point{u*u*u*start.x + 3*u*u*t*a + 3*u*t*t*c + t*t*t*e, u*u*u*start.y + 3*u*u*t*b + 3*u*t*t*d + t*t*t*f})
	}
	return p
}
func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }

var (
	ink   = rgb(7, 15, 28)
	white = rgb(231, 242, 249)
	muted = rgb(133, 157, 181)
	coral = skin{rgb(247, 114, 103), rgb(183, 64, 74), rgb(255, 178, 145), rgb(111, 43, 63)}
	ice   = skin{rgb(127, 219, 237), rgb(55, 139, 172), rgb(206, 249, 252), rgb(37, 83, 116)}
)

type skin struct{ base, shade, light, cuff color.RGBA }
type canvas struct {
	img                          *image.RGBA
	ox, oy, scale, angle, mirror float64
}

func (c canvas) transform(p point) point {
	x, y := p.x*c.scale*c.mirror, p.y*c.scale
	return point{(c.ox + x*math.Cos(c.angle) - y*math.Sin(c.angle)) * samples, (c.oy + x*math.Sin(c.angle) + y*math.Cos(c.angle)) * samples}
}
func (c canvas) fill(p path, col color.RGBA) {
	if len(p) < 3 {
		return
	}
	q := make(path, len(p))
	lo, hi := float64(c.img.Bounds().Max.Y), 0.0
	for i, v := range p {
		q[i] = c.transform(v)
		lo = math.Min(lo, q[i].y)
		hi = math.Max(hi, q[i].y)
	}
	intersections := make([]float64, 0, len(q))
	for y := max(0, int(math.Floor(lo))); y < min(c.img.Bounds().Max.Y, int(math.Ceil(hi))); y++ {
		intersections = intersections[:0]
		fy := float64(y) + .5
		for i, a := range q {
			b := q[(i+1)%len(q)]
			if (a.y <= fy && b.y > fy) || (b.y <= fy && a.y > fy) {
				intersections = append(intersections, a.x+(fy-a.y)*(b.x-a.x)/(b.y-a.y))
			}
		}
		sort.Float64s(intersections)
		for i := 0; i+1 < len(intersections); i += 2 {
			for x := max(0, int(math.Ceil(intersections[i]-.5))); x < min(c.img.Bounds().Max.X, int(math.Ceil(intersections[i+1]-.5))); x++ {
				c.img.SetRGBA(x, y, col)
			}
		}
	}
}

// Round-capped strokes use distance-to-segment coverage on the supersample grid.
func (c canvas) stroke(p path, col color.RGBA, w float64, closed bool) {
	n := len(p) - 1
	if closed {
		n++
	}
	radius := w * c.scale * samples / 2
	for i := 0; i < n; i++ {
		a, b := c.transform(p[i]), c.transform(p[(i+1)%len(p)])
		dx, dy := b.x-a.x, b.y-a.y
		den := dx*dx + dy*dy
		for y := max(0, int(math.Floor(math.Min(a.y, b.y)-radius))); y < min(c.img.Bounds().Max.Y, int(math.Ceil(math.Max(a.y, b.y)+radius))); y++ {
			for x := max(0, int(math.Floor(math.Min(a.x, b.x)-radius))); x < min(c.img.Bounds().Max.X, int(math.Ceil(math.Max(a.x, b.x)+radius))); x++ {
				t := 0.0
				if den > 0 {
					t = math.Max(0, math.Min(1, ((float64(x)+.5-a.x)*dx+(float64(y)+.5-a.y)*dy)/den))
				}
				ex, ey := float64(x)+.5-a.x-t*dx, float64(y)+.5-a.y-t*dy
				if ex*ex+ey*ey <= radius*radius {
					c.img.SetRGBA(x, y, col)
				}
			}
		}
	}
}
func (c canvas) shape(p path, col color.RGBA) { c.fill(p, col); c.stroke(p, ink, 5.5, true) }
func (c canvas) crease(p path)                { c.stroke(p, ink, 2.7, false) }
func (c canvas) highlight(p path)             { c.stroke(p, white, 3, false) }

// wrist is fully framed, rather than cropped by the image edge.
func (c canvas) wrist(s skin) {
	c.shape(move(0, -39).cubic(24, -43, 53, -39, 83, -43).line(91, 62).cubic(59, 56, 24, 59, 0, 53).cubic(-6, 28, -6, -16, 0, -39), s.base)
	c.fill(move(1, 26).line(82, 27).line(89, 60).line(0, 52), s.shade)
	c.shape(move(0, -41).line(28, -42).line(33, 57).line(1, 55).cubic(-6, 31, -6, -17, 0, -41), s.cuff)
	c.stroke(move(12, -29).line(16, 40), s.light, 3, false)
}

// rock: four rounded folded fingers at the leading edge and a separately
// wrapped opposing thumb. The palm, not a boxing-glove oval, carries the mass.
func (c canvas) rock(s skin) {
	c.wrist(s)
	c.shape(move(55, -38).cubic(68, -66, 101, -86, 135, -84).cubic(164, -97, 190, -85, 202, -74).cubic(228, -74, 240, -55, 231, -35).cubic(247, -19, 244, 2, 232, 12).cubic(244, 32, 231, 54, 216, 57).cubic(216, 78, 190, 91, 163, 85).cubic(118, 105, 77, 80, 59, 57).cubic(48, 31, 45, -13, 55, -38), s.base)
	c.fill(move(56, 14).cubic(78, 50, 112, 70, 156, 66).cubic(183, 69, 202, 61, 218, 51).cubic(213, 83, 182, 94, 158, 82).cubic(110, 103, 73, 76, 59, 56), s.shade)
	// Folded phalanges, rounded at the knuckle ends.
	c.shape(move(140, -70).cubic(167, -87, 201, -81, 217, -68).cubic(239, -51, 228, -29, 210, -28).line(168, -27).cubic(150, -32, 136, -48, 140, -70), s.base)
	c.fill(move(152, -67).cubic(174, -78, 199, -72, 212, -62).cubic(194, -66, 179, -60, 160, -52), s.light)
	c.shape(move(165, -28).cubic(185, -38, 217, -34, 229, -22).cubic(244, -6, 230, 15, 212, 15).line(174, 12).cubic(160, 5, 156, -16, 165, -28), s.base)
	c.shape(move(171, 14).cubic(187, 7, 217, 9, 229, 22).cubic(237, 36, 221, 54, 206, 53).line(171, 48).cubic(159, 39, 159, 23, 171, 14), s.base)
	c.shape(move(160, 50).cubic(177, 45, 204, 48, 211, 59).cubic(215, 73, 192, 84, 172, 78).line(151, 70), s.base)
	c.crease(move(203, -16).cubic(210, -19, 217, -18, 222, -15))
	c.crease(move(202, 28).cubic(209, 25, 215, 28, 219, 31))
	// Thumb crosses diagonally in front of the folded digits.
	c.shape(move(81, -37).cubic(94, -57, 115, -59, 132, -42).line(177, 0).cubic(194, 14, 193, 32, 177, 42).cubic(163, 51, 147, 43, 137, 29).line(106, 5).cubic(96, 18, 92, 29, 93, 41).cubic(72, 25, 69, -14, 81, -37), s.base)
	c.fill(move(85, -35).cubic(98, -47, 112, -47, 126, -34).line(163, 0).cubic(149, -3, 133, -12, 118, -22).cubic(104, -32, 96, -24, 84, -15), s.light)
	c.crease(move(142, 13).cubic(149, 9, 163, 13, 171, 22))
	c.crease(move(95, 45).cubic(103, 56, 113, 60, 124, 62))
}

func (c canvas) scissors(s skin) {
	c.wrist(s)
	// A single continuous silhouette: two long separated fingers, no ghost layers.
	c.shape(move(54, -35).cubic(77, -55, 105, -58, 131, -48).line(270, -111).cubic(292, -121, 308, -97, 289, -82).line(190, -20).line(306, 12).cubic(329, 19, 321, 48, 300, 44).line(181, 19).cubic(213, 39, 210, 65, 191, 79).cubic(161, 101, 118, 95, 84, 76).line(57, 58).cubic(48, 28, 46, -10, 54, -35), s.base)
	c.fill(move(63, 26).cubic(100, 66, 153, 78, 192, 60).cubic(197, 86, 154, 102, 119, 87).line(83, 75).line(57, 58), s.shade)
	c.fill(move(139, -44).line(274, -105).cubic(281, -108, 287, -105, 289, -101).line(157, -32), s.light)
	c.stroke(move(195, -10).line(303, 22), s.light, 5, false)
	// Two compact folded digits beneath the V.
	c.shape(move(137, 27).cubic(159, 10, 182, 15, 194, 30).cubic(210, 49, 196, 64, 178, 61).line(147, 54).cubic(135, 49, 130, 35, 137, 27), s.base)
	c.shape(move(133, 57).cubic(148, 45, 177, 50, 186, 66).cubic(192, 82, 175, 90, 157, 84).line(136, 76), s.base)
	c.crease(move(169, 32).line(182, 39))
	c.crease(move(161, 65).line(174, 71))
	// Opposing thumb wraps over the curled ring and little finger.
	c.shape(move(85, -25).cubic(101, -37, 120, -26, 132, -11).line(158, 19).cubic(169, 34, 158, 48, 144, 48).cubic(125, 48, 121, 32, 108, 24).cubic(94, 20, 85, 8, 85, -25), s.base)
	c.fill(move(94, -22).cubic(108, -24, 120, -11, 127, -1).line(146, 21).cubic(124, 13, 110, 1, 94, -6), s.light)
	c.crease(move(132, 23).cubic(139, 19, 146, 23, 151, 28))
	c.crease(move(223, -56).line(233, -44))
	c.crease(move(235, 3).line(231, 15))
	c.crease(move(91, 49).cubic(101, 60, 111, 65, 122, 67))
}

// paper is supported independently: four separated splayed fingers plus an
// opposing thumb. It is never selected by the review animation.
func (c canvas) paper(s skin) {
	c.wrist(s)
	c.shape(move(54, -34).line(102, -62).line(208, -120).cubic(228, -131, 242, -108, 224, -94).line(158, -46).line(276, -81).cubic(300, -88, 309, -62, 285, -53).line(174, -13).line(298, -14).cubic(323, -14, 324, 14, 300, 15).line(178, 21).line(269, 53).cubic(291, 61, 281, 87, 260, 81).line(151, 51).cubic(139, 64, 135, 68, 122, 66).line(161, 111).cubic(176, 130, 154, 149, 137, 132).line(81, 82).cubic(53, 61, 42, 5, 54, -34), s.base)
	c.fill(move(57, 29).cubic(82, 61, 105, 62, 124, 66).line(159, 116).cubic(165, 125, 156, 133, 148, 123).line(81, 81), s.shade)
	c.stroke(move(109, -54).line(212, -110), s.light, 6, false)
	c.crease(move(98, -17).cubic(119, -4, 137, 7, 146, 30))
	c.crease(move(109, 39).cubic(129, 18, 148, 9, 169, 4))
	for _, p := range []path{move(184, -80).line(191, -70), move(234, -64).line(237, -53), move(253, -10).line(253, 9), move(225, 42).line(221, 55)} {
		c.crease(p)
	}
}
func (c canvas) hand(pose string, s skin) {
	switch pose {
	case "rock":
		c.rock(s)
	case "scissors":
		c.scissors(s)
	case "paper":
		c.paper(s)
	default:
		panic("unknown hand pose: " + pose)
	}
}

// Compact embedded uppercase stroke font. Coordinates are vector pen paths,
// not enlarged pixel cells; round caps remain readable after supersampling.
var glyphs = map[rune]string{
	'A': "0,7 2,0 4,7;1,4 3,4", 'B': "0,7 0,0 2.5,0 4,1 4,2.5 2.5,3.5 0,3.5;2.5,3.5 4,4.5 4,6 2.5,7 0,7", 'C': "4,1 3,0 1,0 0,1 0,6 1,7 3,7 4,6", 'D': "0,0 2,0 4,2 4,5 2,7 0,7 0,0", 'E': "4,0 0,0 0,7 4,7;0,3.5 3,3.5", 'F': "4,0 0,0 0,7;0,3.5 3,3.5", 'G': "4,1 3,0 1,0 0,1 0,6 1,7 4,7 4,4 2,4", 'H': "0,0 0,7;4,0 4,7;0,3.5 4,3.5", 'I': "0,0 4,0;2,0 2,7;0,7 4,7", 'J': "4,0 4,6 3,7 1,7 0,6", 'K': "0,0 0,7;4,0 0,4 4,7", 'L': "0,0 0,7 4,7", 'M': "0,7 0,0 2,3 4,0 4,7", 'N': "0,7 0,0 4,7 4,0", 'O': "1,0 3,0 4,1 4,6 3,7 1,7 0,6 0,1 1,0", 'P': "0,7 0,0 3,0 4,1 4,3 3,4 0,4", 'R': "0,7 0,0 3,0 4,1 4,3 3,4 0,4;2,4 4,7", 'S': "4,1 3,0 1,0 0,1 0,2.5 1,3.5 3,3.5 4,4.5 4,6 3,7 1,7 0,6", 'T': "0,0 4,0;2,0 2,7", 'U': "0,0 0,6 1,7 3,7 4,6 4,0", 'V': "0,0 2,7 4,0", 'W': "0,0 1,7 2,4 3,7 4,0", 'Y': "0,0 2,3.5 4,0;2,3.5 2,7", '/': "0,7 4,0", '1': "1,1 2,0 2,7;0,7 4,7", '2': "0,1 1,0 3,0 4,1 4,2 0,7 4,7", '3': "0,0 4,0 2,3;2,3 4,4 4,6 3,7 0,7",
}
var font = buildFont()

func buildFont() map[rune][]path {
	result := make(map[rune][]path)
	for r, data := range glyphs {
		var p path
		var x, y float64
		start := 0
		for i := 0; i <= len(data); i++ {
			if i == len(data) || data[i] == ' ' || data[i] == ';' {
				if i > start {
					if _, err := fmt.Sscanf(data[start:i], "%f,%f", &x, &y); err != nil {
						panic(err)
					}
					p = append(p, point{x, y})
				}
				if i == len(data) || data[i] == ';' {
					result[r] = append(result[r], p)
					p = nil
				}
				start = i + 1
			}
		}
	}
	return result
}
func text(c canvas, label string, cx, y, size float64, col color.RGBA) {
	advance := size * 6
	x := cx - (float64(len(label))*advance-size*2)/2
	for _, r := range label {
		for _, p := range font[r] {
			t := canvas{img: c.img, ox: x, oy: y, scale: size, mirror: 1}
			t.stroke(p, col, .65, false)
		}
		x += advance
	}
}

type frame struct {
	x, y, angle     float64
	reveal, outcome bool
	label           string
	delay           int
}

func ease(t float64) float64 { return t * t * (3 - 2*t) }
func timeline() []frame {
	frames := make([]frame, 0, 58)
	for _, label := range []string{"ROCK", "PAPER", "SCISSORS"} {
		for i := 0; i < 12; i++ {
			t := float64(i) / 11
			y := 0.0
			if t < .42 {
				y = -24 * ease(t/.42)
			} else if t < .78 {
				y = -24 + 39*ease((t-.42)/.36)
			} else {
				y = 15 * (1 - ease((t-.78)/.22))
			}
			frames = append(frames, frame{x: -6 * math.Sin(math.Pi*t), y: y, angle: -.065 * math.Sin(2*math.Pi*t), label: label, delay: 7})
		}
	}
	for i := 0; i < 5; i++ {
		t := ease(float64(i) / 4)
		frames = append(frames, frame{x: -18 * t, y: -8 * t, angle: -.035 * t, label: "READY", delay: 7})
	}
	for i := 0; i < 4; i++ {
		t := ease(float64(i+1) / 4)
		frames = append(frames, frame{x: -18 + 35*t, y: -8 + 8*t, reveal: true, label: "SHOOT", delay: 4})
	}
	for i := 0; i < 5; i++ {
		t := float64(i+1) / 5
		frames = append(frames, frame{x: 17*(1-t) - 7*math.Sin(math.Pi*t), y: 3 * math.Sin(math.Pi*t), reveal: true, outcome: true, label: "ROCK WINS", delay: 6})
	}
	// Identical final images are intentional. Delta encoding makes a completely
	// motionless two-second result economical without any trailing effects.
	for i := 0; i < 8; i++ {
		frames = append(frames, frame{reveal: true, outcome: true, label: "ROCK WINS", delay: 25})
	}
	return frames
}

func background() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width*samples, height*samples))
	for y := 0; y < height*samples; y++ {
		for x := 0; x < width*samples; x++ {
			v := math.Max(0, 1-math.Hypot((float64(x)/samples-360)/520, (float64(y)/samples-195)/360))
			img.SetRGBA(x, y, rgb(uint8(13+6*v), uint8(23+10*v), uint8(40+13*v)))
		}
	}
	return img
}
func render(bg *image.RGBA, f frame) *image.RGBA {
	img := image.NewRGBA(bg.Bounds())
	copy(img.Pix, bg.Pix)
	ui := canvas{img: img, scale: 1, mirror: 1}
	text(ui, "RPS", 60, 24, 1.7, coral.base)
	text(ui, "CLASH REPLAY", 596, 25, 1.4, muted)
	ui.stroke(move(32, 51).line(688, 51), rgb(43, 60, 80), 1, false)
	text(ui, f.label, 360, 68, 2.7, white)
	pose := "rock"
	if f.reveal {
		pose = "scissors"
	}
	left := canvas{img: img, ox: 53 + f.x, oy: 222 + f.y, scale: 1.02, angle: f.angle, mirror: 1}
	right := canvas{img: img, ox: 667 - f.x, oy: 222 + f.y, scale: .90, angle: -f.angle, mirror: -1}
	left.hand("rock", coral)
	right.hand(pose, ice)
	ui.stroke(move(32, 338).line(688, 338), rgb(43, 60, 80), 1, false)
	if f.outcome {
		text(ui, "ROCK", 177, 355, 2.2, coral.light)
		text(ui, "SCISSORS", 543, 355, 2.2, ice.light)
		text(ui, "WINNER", 177, 381, 1.1, coral.base)
		text(ui, "PLAYER 2", 543, 381, 1.1, muted)
	} else {
		text(ui, "PLAYER 1", 177, 363, 1.5, coral.light)
		text(ui, "PLAYER 2", 543, 363, 1.5, ice.light)
	}
	return img
}

// A fixed palette keeps static pixels identical throughout the loop. Ramp
// colors preserve antialiased edges without animated error-diffusion noise.
func makePalette() color.Palette {
	p := color.Palette{color.RGBA{0, 0, 0, 0}}
	add := func(c color.RGBA) { p = append(p, c) }
	for i := 0; i < 16; i++ {
		v := float64(i) / 15
		add(rgb(uint8(13+6*v), uint8(23+10*v), uint8(40+13*v)))
	}
	bases := []color.RGBA{ink, white, muted, coral.base, coral.shade, coral.light, coral.cuff, ice.base, ice.shade, ice.light, ice.cuff, rgb(43, 60, 80)}
	for _, c := range bases {
		add(c)
	}
	for _, c := range bases {
		for _, other := range []color.RGBA{ink, rgb(17, 30, 49)} {
			for i := 1; i <= 7; i++ {
				t := float64(i) / 8
				add(rgb(uint8(float64(c.R)*(1-t)+float64(other.R)*t), uint8(float64(c.G)*(1-t)+float64(other.G)*t), uint8(float64(c.B)*(1-t)+float64(other.B)*t)))
			}
		}
	}
	for _, s := range []skin{coral, ice} {
		for _, c := range []color.RGBA{s.shade, s.light} {
			for i := 1; i <= 7; i++ {
				t := float64(i) / 8
				add(rgb(uint8(float64(s.base.R)*(1-t)+float64(c.R)*t), uint8(float64(s.base.G)*(1-t)+float64(c.G)*t), uint8(float64(s.base.B)*(1-t)+float64(c.B)*t)))
			}
		}
	}
	return p
}
func quantize(src *image.RGBA, p color.Palette, cache map[uint32]uint8) *image.Paletted {
	dst := image.NewPaletted(image.Rect(0, 0, width, height), p)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b := 0, 0, 0
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					i := src.PixOffset(x*samples+sx, y*samples+sy)
					r += int(src.Pix[i])
					g += int(src.Pix[i+1])
					b += int(src.Pix[i+2])
				}
			}
			r /= samples * samples
			g /= samples * samples
			b /= samples * samples
			key := uint32(r)<<16 | uint32(g)<<8 | uint32(b)
			index, ok := cache[key]
			if !ok {
				best := int(^uint(0) >> 1)
				for i := 1; i < len(p); i++ {
					c := p[i].(color.RGBA)
					dr, dg, db := r-int(c.R), g-int(c.G), b-int(c.B)
					d := dr*dr + dg*dg + db*db
					if d < best {
						best = d
						index = uint8(i)
					}
				}
				cache[key] = index
			}
			dst.SetColorIndex(x, y, index)
		}
	}
	return dst
}

// Delta rectangles retain two visible hands, not ghost images: every changed
// pixel (including exposed background) is replaced; unchanged pixels persist.
func delta(previous, current *image.Paletted) *image.Paletted {
	box := image.Rectangle{Min: image.Pt(width, height), Max: image.Pt(0, 0)}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if previous.ColorIndexAt(x, y) != current.ColorIndexAt(x, y) {
				box.Min.X = min(box.Min.X, x)
				box.Min.Y = min(box.Min.Y, y)
				box.Max.X = max(box.Max.X, x+1)
				box.Max.Y = max(box.Max.Y, y+1)
			}
		}
	}
	if box.Empty() {
		box = image.Rect(0, 0, 1, 1)
	}
	dst := image.NewPaletted(box, current.Palette)
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			if previous.ColorIndexAt(x, y) != current.ColorIndexAt(x, y) {
				dst.SetColorIndex(x, y, current.ColorIndexAt(x, y))
			}
		}
	}
	return dst
}
func generateReview(filename string) error {
	frames := timeline()
	p := makePalette()
	cache := make(map[uint32]uint8)
	bg := background()
	out := &gif.GIF{LoopCount: 0, Config: image.Config{ColorModel: p, Width: width, Height: height}}
	var previous *image.Paletted
	duration := 0
	for _, f := range frames {
		full := quantize(render(bg, f), p, cache)
		encoded := full
		if previous != nil {
			encoded = delta(previous, full)
		}
		out.Image = append(out.Image, encoded)
		out.Delay = append(out.Delay, f.delay)
		out.Disposal = append(out.Disposal, gif.DisposalNone)
		previous = full
		duration += f.delay
	}
	// Encode to a sibling temporary file and rename only after close succeeds.
	file, err := os.CreateTemp(filepath.Dir(filename), ".review-*")
	if err != nil {
		return fmt.Errorf("create review temporary file: %w", err)
	}
	temp := file.Name()
	defer os.Remove(temp)
	if err = gif.EncodeAll(file, out); err != nil {
		file.Close()
		return fmt.Errorf("encode review: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("close review: %w", err)
	}
	if err = os.Chmod(temp, 0644); err != nil {
		return fmt.Errorf("set review permissions: %w", err)
	}
	if err = os.Rename(temp, filename); err != nil {
		return fmt.Errorf("publish review: %w", err)
	}
	info, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("stat review: %w", err)
	}
	fmt.Printf("%s\n%d x %d | %d frames | %.2f seconds | %d colors | %d bytes (%.2f MiB) | infinite loop\n3x supersampling | ROCK vs SCISSORS | stable final hold: 2.00 seconds\n", filename, width, height, len(frames), float64(duration)/100, len(p), info.Size(), float64(info.Size())/(1024*1024))
	return nil
}
func run(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: gifpoc [output-directory]")
	}
	dir := "/var/www/jediui.com/public/replays"
	if len(args) == 1 {
		dir = args[0]
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	return generateReview(filepath.Join(dir, "review.gif"))
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gifpoc:", err)
		os.Exit(1)
	}
}
