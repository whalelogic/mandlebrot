// Package palette provides color palettes and interpolation functions.
package palette

import (
	"image/color"
	"sort"
)

// Color holds a position (Step 0..1) and a color.
// If Step is zero for multiple entries, Normalize will evenly distribute.
type Color struct {
	Step  float64
	Color color.Color
}

type ColorMap struct {
	Keyword string
	Colors  []Color
}

// ColorPalettes contains palettes you can choose from. All steps should ideally be in range [0,1].
// If some entries have Step==0 they will be normalized at runtime by Normalize().
var ColorPalettes = []ColorMap{
	{"NebulaSpectre", []Color{
		{0.0,  color.RGBA{0x09, 0x04, 0x20, 0xff}}, // deep violet
		{0.15, color.RGBA{0x3A, 0x0F, 0x73, 0xff}}, // purple
		{0.35, color.RGBA{0x8D, 0x1A, 0xA8, 0xff}}, // magenta
		{0.55, color.RGBA{0xE7, 0x36, 0x7F, 0xff}}, // hot pink
		{0.75, color.RGBA{0x3B, 0xD6, 0xC2, 0xff}}, // cyan–teal
		{1.0,  color.RGBA{0xF0, 0xFF, 0xFF, 0xff}}, // bright highlight
	}},

	{"MonochromeSlate", []Color{
		{0.0, color.RGBA{0x00, 0x00, 0x00, 0xff}},
		{0.5, color.RGBA{0x70, 0x70, 0x70, 0xff}},
		{1.0, color.RGBA{0xff, 0xff, 0xff, 0xff}},
	}},

	{"MetallicChrome", []Color{
		{0.0, color.RGBA{0x06, 0x0b, 0x14, 0xff}},
		{0.2, color.RGBA{0x3a, 0x3f, 0x45, 0xff}},
		{0.45, color.RGBA{0x9e, 0xae, 0xb4, 0xff}},
		{0.7, color.RGBA{0xe7, 0xd8, 0xb0, 0xff}},
		{1.0, color.RGBA{0xff, 0xff, 0xff, 0xff}},
	}},

	{"ThermalHeat", []Color{
		{0.0, color.RGBA{0x00, 0x00, 0x00, 0xff}},
		{0.25, color.RGBA{0x70, 0x00, 0x00, 0xff}},
		{0.5, color.RGBA{0xff, 0x40, 0x00, 0xff}},
		{0.75, color.RGBA{0xff, 0xd0, 0x00, 0xff}},
		{1.0, color.RGBA{0xff, 0xff, 0xff, 0xff}},
	}},

	{"AuroraArc", []Color{
		{0.0, color.RGBA{0x01, 0x13, 0x1f, 0xff}},
		{0.2, color.RGBA{0x03, 0x6b, 0x5f, 0xff}},
		{0.45, color.RGBA{0x54, 0xe6, 0xb2, 0xff}},
		{0.7, color.RGBA{0x95, 0x43, 0xd6, 0xff}},
		{1.0, color.RGBA{0xf8, 0xf9, 0xff, 0xff}},
	}},
}

// Get returns the ColorMap by keyword (case-sensitive) or nil if not found.
func Get(keyword string) *ColorMap {
	for i := range ColorPalettes {
		if ColorPalettes[i].Keyword == keyword {
			// return a copy so callers can mutate returned Colors/normalize safely
			cpy := ColorPalettes[i]
			Normalize(&cpy)
			return &cpy
		}
	}
	return nil
}

// Normalize fills in missing Step values (Step == 0) by evenly spacing them.
// It also ensures first and last steps are 0 and 1 respectively if they are unspecified.
func Normalize(cm *ColorMap) {
	if cm == nil || len(cm.Colors) == 0 {
		return
	}

	// If every Color has a non-zero Step, just sort and clamp.
	allSpecified := true
	for _, c := range cm.Colors {
		if c.Step == 0 {
			allSpecified = false
			break
		}
	}
	if allSpecified {
		sort.Slice(cm.Colors, func(i, j int) bool { return cm.Colors[i].Step < cm.Colors[j].Step })
		// clamp to [0,1]
		for i := range cm.Colors {
			if cm.Colors[i].Step < 0 {
				cm.Colors[i].Step = 0
			}
			if cm.Colors[i].Step > 1 {
				cm.Colors[i].Step = 1
			}
		}
		return
	}

	// Otherwise evenly distribute across length, but respect any non-zero Steps.
	n := len(cm.Colors)
	// Build indices with fixed steps
	type idxStep struct {
		idx  int
		step float64
	}
	var fixed []idxStep
	for i, c := range cm.Colors {
		if c.Step > 0 {
			if c.Step < 0 {
				c.Step = 0
			}
			if c.Step > 1 {
				c.Step = 1
			}
			fixed = append(fixed, idxStep{i, c.Step})
		}
	}
	// If no fixed points, evenly space from 0..1
	if len(fixed) == 0 {
		for i := range cm.Colors {
			cm.Colors[i].Step = float64(i) / float64(n-1)
		}
		return
	}
	// Ensure first and last are fixed at 0 and 1
	if fixed[0].idx != 0 {
		fixed = append([]idxStep{{0, 0.0}}, fixed...)
		cm.Colors[0].Step = 0
	}
	if fixed[len(fixed)-1].idx != n-1 {
		fixed = append(fixed, idxStep{n - 1, 1.0})
		cm.Colors[n-1].Step = 1
	}
	// fill between fixed pairs
	for k := 0; k < len(fixed)-1; k++ {
		a := fixed[k]
		b := fixed[k+1]
		ia, ib := a.idx, b.idx
		stepspan := b.step - a.step
		spanCount := float64(ib - ia)
		for i := ia; i <= ib; i++ {
			if i == ia {
				cm.Colors[i].Step = a.step
				continue
			}
			frac := float64(i-ia) / spanCount
			cm.Colors[i].Step = a.step + frac*stepspan
		}
	}
	// finally sort by step
	sort.Slice(cm.Colors, func(i, j int) bool { return cm.Colors[i].Step < cm.Colors[j].Step })
}

// Interpolate returns an interpolated color for t in [0,1] across the ColorMap.
// If t <= first step returns first color, if t >= last returns last.
func (cm *ColorMap) Interpolate(t float64) color.RGBA {
	if cm == nil || len(cm.Colors) == 0 {
		return color.RGBA{0, 0, 0, 0xff}
	}
	if t <= 0 {
		return toRGBA(cm.Colors[0].Color)
	}
	if t >= 1 {
		return toRGBA(cm.Colors[len(cm.Colors)-1].Color)
	}

	// find interval
	for i := 0; i < len(cm.Colors)-1; i++ {
		a := cm.Colors[i]
		b := cm.Colors[i+1]
		if t >= a.Step && t <= b.Step {
			segT := (t - a.Step) / (b.Step - a.Step)
			return lerpRGBA(toRGBA(a.Color), toRGBA(b.Color), segT)
		}
	}
	// fallback
	return toRGBA(cm.Colors[len(cm.Colors)-1].Color)
}

// toRGBA converts a color.Color to color.RGBA (with premultiplied alpha normalized).
func toRGBA(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	// color.Color returns values in 0..65535, convert to 0..255
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

// lerpRGBA linearly interpolates between two RGBA colors in sRGB space.
func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	return color.RGBA{
		uint8(clamp((1-t)*float64(a.R)+t*float64(b.R), 0, 255)),
		uint8(clamp((1-t)*float64(a.G)+t*float64(b.G), 0, 255)),
		uint8(clamp((1-t)*float64(a.B)+t*float64(b.B), 0, 255)),
		uint8(clamp((1-t)*float64(a.A)+t*float64(b.A), 0, 255)),
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

