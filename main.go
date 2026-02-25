package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"os"
)

type RawImage struct {
	Data   []uint8
	Stride int
	Width  int
	Height int
}

func (r *RawImage) GetRGB(x, y int) (int, int, int) {
	startPos := y*r.Stride + x*4
	return int(r.Data[startPos]), int(r.Data[startPos+1]), int(r.Data[startPos+2])
}

func (r *RawImage) toPNG(filename string) error {
	img := image.NewRGBA(image.Rect(0, 0, r.Width, r.Height))

	for y := 0; y < r.Height; y++ {

		srcStart := y * r.Stride
		srcEnd := srcStart + (r.Width * 4)

		dstStart := y * img.Stride

		copy(img.Pix[dstStart:dstStart+(r.Width*4)], r.Data[srcStart:srcEnd])
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

func LoadImage(path string) (*RawImage, error) {
	imgFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening image file: %w", err)
	}
	defer imgFile.Close()

	srcImg, _, err := image.Decode(imgFile)
	if err != nil {
		return nil, fmt.Errorf("error decoding image file: %w", err)
	}

	bounds := srcImg.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	dstImg := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dstImg, dstImg.Bounds(), srcImg, bounds.Min, draw.Src)

	return &RawImage{
		Data:   dstImg.Pix,
		Stride: (dstImg.Stride),
		Width:  (width),
		Height: (height),
	}, nil
}

func CalculateEnergy(image *RawImage) []int {
	energyMap := make([]int, image.Width*image.Height)
	for y := 0; y < image.Height; y++ {
		for x := 0; x < image.Width; x++ {

			leftX, rightX := x-1, x+1
			if x == 0 {
				leftX = x
			}
			if x == image.Width-1 {
				rightX = x
			}

			topY, botY := y-1, y+1
			if y == 0 {
				topY = y
			}
			if y == image.Height-1 {
				botY = y
			}

			r1, g1, b1 := image.GetRGB(leftX, y)
			r2, g2, b2 := image.GetRGB(rightX, y)
			rx, gx, bx := r2-r1, g2-g1, b2-b1
			dx := rx*rx + gx*gx + bx*bx

			r1, g1, b1 = image.GetRGB(x, topY)
			r2, g2, b2 = image.GetRGB(x, botY)
			ry, gy, by := r2-r1, g2-g1, b2-b1
			dy := ry*ry + gy*gy + by*by

			energyMap[y*image.Width+x] = dx + dy
		}
	}
	return energyMap
}

func CalculateAndRemoveSeam(image *RawImage, energyMap []int) {
	width, height := image.Width, image.Height

	energy := make([]int, len(energyMap))
	prevMinIndex := make([]int, len(energyMap))
	copy(energy, energyMap)

	// Calculate Seam
	// Cal actual Energy
	for y := 1; y < height; y++ {
		for x := 0; x < width; x++ {
			prevMin := 0
			minValue := energy[y*width+x+prevMin]

			if x > 0 && energy[y*width+x-1] < minValue {
				prevMin = -1
				minValue = energy[y*width+x+prevMin]
			}
			if x < width-1 && energy[y*width+x+1] < minValue {
				prevMin = 1
				minValue = energy[y*width+x+prevMin]
			}

			energy[y*width+x] += minValue
			prevMinIndex[y*width+x] = prevMin
		}
	}
	// Cal least energy
	minXValue := energy[(height-1)*width]
	curX := 0
	for x := 1; x < width; x++ {
		if energy[(height-1)*width+x] < minXValue {
			minXValue = energy[(height-1)*width+x]
			curX = x
		}
	}
	// Backtrack & Remove Seam
	xPos := curX
	for y := height - 1; y >= 0; y-- {
		for x := xPos; x < width-1; x++ {
			startPos := y*image.Stride + x*4

			image.Data[startPos+0] = image.Data[startPos+4+0]
			image.Data[startPos+1] = image.Data[startPos+4+1]
			image.Data[startPos+2] = image.Data[startPos+4+2]
			image.Data[startPos+3] = image.Data[startPos+4+3]
		}
		xPos = xPos + prevMinIndex[y*width+xPos]
	}

	image.Width--

}

func main() {

	image, err := LoadImage("test.jpg")
	if err != nil {
		fmt.Println(err.Error())
	}

	for i := 0; i < 500; i++ {
		fmt.Println(i)

		energyMap := CalculateEnergy(image)
		CalculateAndRemoveSeam(image, energyMap)
	}

	image.toPNG("output.png")
}
