package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

var (
	colorStep     float64 = 6000
	xpos, ypos    float64 = -0.00275, 0.78912
	width, height int     = 2048, 2048
	maxIteration  int     = 800
	escapeRadius  float64 = .125689
	filename      string  = "mandelbrot.png"
)

var waitGroup sync.WaitGroup

func main() {
	//override defaults if env var exists (os.LookupEnv)
	colorStep = getEnvF("COLORSTEP", colorStep)
	xpos = getEnvF("XPOS", xpos)
	ypos = getEnvF("YPOS", ypos)
	width = getEnvI("WIDTH", width)
	height = getEnvI("HEIGHT", height)
	maxIteration = getEnvI("MAXITERATION", maxIteration)
	escapeRadius = getEnvF("ESCAPERADIUS", escapeRadius)
	filename = getEnvS("FILENAME", filename)

	if amIRunningOnLambda() {
		filename = "/tmp/" + filename
		lambda.Start(HandleRequest)

	} else {
		runBrot()

	}
}

func HandleRequest() {
	runBrot()
}

func runBrot() {
	start := time.Now()
	defer func() {
		fmt.Printf("rendering took %v\n", time.Since(start))
	}()

	done := make(chan struct{})
	ticker := time.NewTicker(time.Millisecond * 500)

	go func() {
		for {
			select {
			case <-ticker.C:
				//fmt.Printf("# of running goroutines: %d\n", runtime.NumGoroutine())

			case <-done:
				ticker.Stop()
				fmt.Printf("\n\nMandelbrot set rendered into `%s`\n", filename)
			}
		}
	}()

	if colorStep < float64(maxIteration) {
		colorStep = float64(maxIteration)
	}
	colors := interpolateColors(colorStep)

	if len(colors) > 0 {
		render(maxIteration, colors, done)
	}
	time.Sleep(time.Second)
}

func interpolateColors(numberOfColors float64) []color.RGBA {
	var factor float64
	steps := []float64{}
	cols := []uint32{}
	interpolated := []uint32{}
	interpolatedColors := []color.RGBA{}

	factor = 1.0 / numberOfColors

	for index, col := range lsdcolors {
		if col.Step == 0.0 && index != 0 {
			stepRatio := float64(index+1) / float64(len(lsdcolors))
			step := float64(int(stepRatio*100)) / 100 // truncate to 2 decimal precision
			steps = append(steps, step)
		} else {
			steps = append(steps, col.Step)
		}
		r, g, b, a := col.Color.RGBA()
		r /= 0xff
		g /= 0xff
		b /= 0xff
		a /= 0xff
		uintColor := uint32(r)<<24 | uint32(g)<<16 | uint32(b)<<8 | uint32(a)
		cols = append(cols, uintColor)
	}

	var min, max, minColor, maxColor float64
	if len(lsdcolors) == len(steps) && len(lsdcolors) == len(cols) {
		for i := 0.0; i <= 1; i += factor {
			for j := 0; j < len(lsdcolors)-1; j++ {
				if i >= steps[j] && i < steps[j+1] {
					min = steps[j]
					max = steps[j+1]
					minColor = float64(cols[j])
					maxColor = float64(cols[j+1])
					uintColor := cosineInterpolation(maxColor, minColor, (i-min)/(max-min))
					interpolated = append(interpolated, uint32(uintColor))
				}
			}
		}
	}

	for _, pixelValue := range interpolated {
		r := pixelValue >> 24 & 0xff
		g := pixelValue >> 16 & 0xff
		b := pixelValue >> 8 & 0xff
		a := 0xff

		interpolatedColors = append(interpolatedColors, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
	}

	return interpolatedColors
}

func render(maxIteration int, colors []color.RGBA, done chan struct{}) {
	ratio := float64(height) / float64(width)
	xmin, xmax := xpos-escapeRadius/2.0, math.Abs(xpos+escapeRadius/2.0)
	ymin, ymax := ypos-escapeRadius*ratio/2.0, math.Abs(ypos+escapeRadius*ratio/2.0)

	image := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{width, height}})

	for iy := 0; iy < height; iy++ {
		waitGroup.Add(1)
		go func(iy int) {
			defer waitGroup.Done()

			for ix := 0; ix < width; ix++ {
				var x = xmin + (xmax-xmin)*float64(ix)/float64(width-1)
				var y = ymin + (ymax-ymin)*float64(iy)/float64(height-1)
				norm, it := mandelIteration(x, y, maxIteration)
				iteration := float64(maxIteration-it) + math.Log(norm)

				if int(math.Abs(iteration)) < len(colors)-1 {
					color1 := colors[int(math.Abs(iteration))]
					color2 := colors[int(math.Abs(iteration))+1]
					color := linearInterpolation(rgbaToUint(color1), rgbaToUint(color2), uint32(iteration))

					image.Set(ix, iy, uint32ToRgba(color))
				}
			}
		}(iy)
	}

	waitGroup.Wait()

	output, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	} else {
		png.Encode(output, image)
	}
	done <- struct{}{}
}

func cosineInterpolation(c1, c2, mu float64) float64 {
	mu2 := (1 - math.Cos(mu*math.Pi)) / 2.0
	return c1*(1-mu2) + c2*mu2
}

func linearInterpolation(c1, c2, mu uint32) uint32 {
	return c1*(1-mu) + c2*mu
}

func mandelIteration(cx, cy float64, maxIter int) (float64, int) {
	var x, y, xx, yy float64 = 0.0, 0.0, 0.0, 0.0

	for i := 0; i < maxIter; i++ {
		xy := x * y
		xx = x * x
		yy = y * y
		if xx+yy > 4 {
			return xx + yy, i
		}
		x = xx - yy + cx
		y = 2*xy + cy
	}

	logZn := (x*x + y*y) / 2
	return logZn, maxIter
}

func rgbaToUint(color color.RGBA) uint32 {
	r, g, b, a := color.RGBA()
	r /= 0xff
	g /= 0xff
	b /= 0xff
	a /= 0xff
	return uint32(r)<<24 | uint32(g)<<16 | uint32(b)<<8 | uint32(a)
}

func uint32ToRgba(col uint32) color.RGBA {
	r := col >> 24 & 0xff
	g := col >> 16 & 0xff
	b := col >> 8 & 0xff
	a := 0xff
	return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
}

type Color struct {
	Step  float64
	Color color.Color
}

type ColorMap struct {
	Keyword string
	Colors  []Color
}

var lsdcolors = []Color{
	{Color: color.RGBA{0x00, 0x04, 0x0f, 0xff}},
	{Color: color.RGBA{0x03, 0x26, 0x28, 0xff}},
	{Color: color.RGBA{0x07, 0x3e, 0x1e, 0xff}},
	{Color: color.RGBA{0x18, 0x55, 0x08, 0xff}},
	{Color: color.RGBA{0x5f, 0x6e, 0x0f, 0xff}},
	{Color: color.RGBA{0x84, 0x50, 0x19, 0xff}},
	{Color: color.RGBA{0x9b, 0x30, 0x22, 0xff}},
	{Color: color.RGBA{0xb4, 0x92, 0x2f, 0xff}},
	{Color: color.RGBA{0x94, 0xca, 0x3d, 0xff}},
	{Color: color.RGBA{0x4f, 0xd5, 0x51, 0xff}},
	{Color: color.RGBA{0x66, 0xff, 0xb3, 0xff}},
	{Color: color.RGBA{0x82, 0xc9, 0xe5, 0xff}},
	{Color: color.RGBA{0x9d, 0xa3, 0xeb, 0xff}},
	{Color: color.RGBA{0xd7, 0xb5, 0xf3, 0xff}},
	{Color: color.RGBA{0xfd, 0xd6, 0xf6, 0xff}},
	{Color: color.RGBA{0xff, 0xf0, 0xf2, 0xff}},
}
