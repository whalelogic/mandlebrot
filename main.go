package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"runtime"
	"sync"

	"github.com/whalelogic/mandelbrot/palette"
)

func main() {
	// Flags for the renderer
	width := flag.Int("width", 1600, "output image width in pixels")
	height := flag.Int("height", 1200, "output image height in pixels")
	xmin := flag.Float64("xmin", -2.2, "left x coordinate")
	xmax := flag.Float64("xmax", 1.0, "right x coordinate")
	ymin := flag.Float64("ymin", -1.6, "bottom y coordinate")
	ymax := flag.Float64("ymax", 1.6, "top y coordinate")
	iters := flag.Int("iters", 1200, "max iteration count")
	outfile := flag.String("outfile", "mandelbrot.png", "output PNG filename")
	pal := flag.String("palette", "NebulaSpectre", "palette name (case-sensitive)")
	concurrency := flag.Int("procs", runtime.NumCPU(), "concurrent worker count")
	smooth := flag.Bool("smooth", true, "use smooth coloring (continuous escape-time)")
	flag.Parse()

	runtime.GOMAXPROCS(*concurrency)

	cmap := palette.Get(*pal)
	if cmap == nil {
		fmt.Fprintf(os.Stderr, "palette %q not found. Available palettes:\n", *pal)
		for _, p := range palette.ColorPalettes {
			fmt.Fprintf(os.Stderr, "  - %s\n", p.Keyword)
		}
		os.Exit(2)
	}
	// ensure palette normalized
	palette.Normalize(cmap)

	img := image.NewRGBA(image.Rect(0, 0, *width, *height))

	// Worker pattern to compute rows in parallel.
	rows := make(chan int, *height)
	var wg sync.WaitGroup
	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for y := range rows {
				computeRow(img, y, *width, *height, *xmin, *xmax, *ymin, *ymax, *iters, cmap, *smooth)
			}
		}()
	}

	for y := 0; y < *height; y++ {
		rows <- y
	}
	close(rows)
	wg.Wait()

	// Save file
	f, err := os.Create(*outfile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode png: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Saved %s (%dx%d) using palette %s\n", *outfile, *width, *height, *pal)
}

// computeRow computes a single row y and writes pixels into img.
func computeRow(img *image.RGBA, y, width, height int, xmin, xmax, ymin, ymax float64, iters int, cmap *palette.ColorMap, smooth bool) {
	for x := 0; x < width; x++ {
		// map pixel to complex plane
		cre := xmin + (float64(x)/float64(width))*(xmax-xmin)
		cim := ymin + (float64(y)/float64(height))*(ymax-ymin)
		c := complex(cre, cim)

		iter, z := mandelbrotIterations(c, iters)
		var t float64
		if iter >= iters {
			// inside set -> black (or the palette start)
			t = 0.0
		} else {
			if smooth {
				// continuous (smooth) iteration count:
				// nu = n + 1 - log(log|z|)/log(2)
				// normalize by iters to map to palette
				mag := cmplxAbs(z)
				if mag <= 0 {
					mag = 1e-16
				}
				nu := float64(iter) + 1 - math.Log(math.Log(mag))/math.Log(2)
				// nu might be <0 if weird; clamp
				if nu < 0 {
					nu = float64(iter)
				}
				t = nu / float64(iters)
			} else {
				t = float64(iter) / float64(iters)
			}
			// optionally warp t for aesthetic (gamma)
			// t = math.Pow(t, 0.8)
		}

		clr := cmap.Interpolate(t)
		img.SetRGBA(x, y, clr)
	}
}

// mandelbrotIterations calculates standard escape-time iterations and returns (n, z_n)
// z_n is the last computed z value (useful for smoothing).
func mandelbrotIterations(c complex128, maxIter int) (int, complex128) {
	var z complex128
	for n := 0; n < maxIter; n++ {
		z = z*z + c
		if real(z)*real(z)+imag(z)*imag(z) > 4.0 {
			return n, z
		}
	}
	return maxIter, z
}

// cmplxAbs returns the magnitude of a complex128.
func cmplxAbs(z complex128) float64 {
	return math.Hypot(real(z), imag(z))
}

