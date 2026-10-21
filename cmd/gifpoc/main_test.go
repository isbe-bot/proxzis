package main

import (
	"bytes"
	"image"
	"image/gif"
	"testing"
)

func TestTimeline(t *testing.T) {
	frames := timeline()
	duration := 0
	counts := map[string]int{}
	if len(frames) < 40 || len(frames) > 70 {
		t.Fatalf("frame count: %d", len(frames))
	}
	for _, f := range frames {
		duration += f.delay
		counts[f.label]++
		if f.delay <= 0 {
			t.Fatal("nonpositive delay")
		}
		if !f.reveal && f.outcome {
			t.Fatal("premature outcome")
		}
	}
	for _, label := range []string{"ROCK", "PAPER", "SCISSORS"} {
		if counts[label] != 12 {
			t.Fatalf("%s beat: %d frames", label, counts[label])
		}
	}
	if duration < 500 || duration > 600 {
		t.Fatalf("duration: %d cs", duration)
	}
	last := frames[len(frames)-1]
	hold := 0
	for _, f := range frames[len(frames)-8:] {
		if f != last {
			t.Fatal("final pose must be absolutely stable")
		}
		hold += f.delay
	}
	if hold != 200 {
		t.Fatalf("final hold: %d cs", hold)
	}
}
func TestPalette(t *testing.T) {
	p := makePalette()
	if len(p) > 256 {
		t.Fatalf("palette overflow: %d", len(p))
	}
	_, _, _, a := p[0].RGBA()
	if a != 0 {
		t.Fatal("delta transparency missing")
	}
}
func TestAllPosesFramedAndDistinct(t *testing.T) {
	renders := map[string][]byte{}
	for _, pose := range []string{"rock", "paper", "scissors"} {
		img := image.NewRGBA(image.Rect(0, 0, width*samples, height*samples))
		c := canvas{img: img, ox: 53, oy: 210, scale: 1.02, mirror: 1}
		c.hand(pose, coral)
		minX, minY, maxX, maxY := width*samples, height*samples, 0, 0
		for y := 0; y < img.Bounds().Max.Y; y++ {
			for x := 0; x < img.Bounds().Max.X; x++ {
				if img.RGBAAt(x, y).A != 0 {
					minX = min(minX, x)
					minY = min(minY, y)
					maxX = max(maxX, x)
					maxY = max(maxY, y)
				}
			}
		}
		if minX < 25*samples || maxX > 390*samples || minY < 65*samples || maxY > 365*samples {
			t.Fatalf("%s bounds: (%d,%d)-(%d,%d)", pose, minX/samples, minY/samples, maxX/samples, maxY/samples)
		}
		if maxX-minX < 200*samples {
			t.Fatalf("%s hand too small", pose)
		}
		for other, pix := range renders {
			if bytes.Equal(pix, img.Pix) {
				t.Fatalf("%s identical to %s", pose, other)
			}
		}
		renders[pose] = img.Pix
	}
}
func TestDeltaReconstructsWithoutGhosts(t *testing.T) {
	p := makePalette()
	a := image.NewPaletted(image.Rect(0, 0, width, height), p)
	b := image.NewPaletted(a.Bounds(), p)
	for i := range a.Pix {
		a.Pix[i] = 1
		b.Pix[i] = 1
	}
	a.SetColorIndex(30, 40, 20)
	b.SetColorIndex(45, 45, 25)
	d := delta(a, b)
	for y := d.Rect.Min.Y; y < d.Rect.Max.Y; y++ {
		for x := d.Rect.Min.X; x < d.Rect.Max.X; x++ {
			if v := d.ColorIndexAt(x, y); v != 0 {
				a.SetColorIndex(x, y, v)
			}
		}
	}
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("delta leaves old geometry behind")
	}
	if got := delta(b, b).Bounds(); got != image.Rect(0, 0, 1, 1) {
		t.Fatalf("stable frame bounds: %v", got)
	}
	// The encoded stream must preserve disposal and transparent delta pixels.
	out := &gif.GIF{Image: []*image.Paletted{b, delta(b, b)}, Delay: []int{7, 25}, Disposal: []byte{gif.DisposalNone, gif.DisposalNone}, Config: image.Config{ColorModel: p, Width: width, Height: height}}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, out); err != nil {
		t.Fatal(err)
	}
	decoded, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Image) != 2 || decoded.Disposal[1] != gif.DisposalNone {
		t.Fatal("GIF metadata changed")
	}
}
func TestEmbeddedFontCoverage(t *testing.T) {
	for _, s := range []string{"RPS CLASH REPLAY ROCK PAPER SCISSORS READY SHOOT ROCK WINS WINNER PLAYER 1 PLAYER 2"} {
		for _, r := range s {
			if r != ' ' && len(font[r]) == 0 {
				t.Fatalf("missing glyph %q", r)
			}
		}
	}
}
