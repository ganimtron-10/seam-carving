package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

type RawImage struct {
	Data   []uint8
	Stride int
	Width  int
	Height int
}

func (r *RawImage) toImageFile(filename string) error {
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

	// return png.Encode(f, img)
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
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

func CalculateEnergy(image *RawImage, energyMap []int) []int {
	for y := 1; y < image.Height-1; y++ {
		rowIdx := y * image.Stride
		prevRow := (y - 1) * image.Stride
		nextRow := (y + 1) * image.Stride
		mapOff := y * image.Width

		for x := 0; x < image.Width; x++ {

			leftX, rightX := x-1, x+1
			if x == 0 {
				leftX = x
			}
			if x == image.Width-1 {
				rightX = x
			}

			leftXPos := rowIdx + leftX*4
			rightXPos := rowIdx + rightX*4
			r1, g1, b1 := image.Data[leftXPos], image.Data[leftXPos+1], image.Data[leftXPos+2]
			r2, g2, b2 := image.Data[rightXPos], image.Data[rightXPos+1], image.Data[rightXPos+2]
			rx, gx, bx := int(r2-r1), int(g2-g1), int(b2-b1)
			dx := rx*rx + gx*gx + bx*bx

			topYPos := prevRow + x*4
			botYPos := nextRow + x*4
			r1, g1, b1 = image.Data[topYPos], image.Data[topYPos+1], image.Data[topYPos+2]
			r2, g2, b2 = image.Data[botYPos], image.Data[botYPos+1], image.Data[botYPos+2]
			ry, gy, by := int(r2-r1), int(g2-g1), int(b2-b1)
			dy := ry*ry + gy*gy + by*by

			energyMap[mapOff+x] = dx + dy
		}
	}
	return energyMap
}

func CalculateAndRemoveSeam(image *RawImage, energyMap, cumulativeEnergy, seam []int, prevMinIndex []int8) {
	width, height := image.Width, image.Height

	copy(cumulativeEnergy, energyMap)

	// Calculate Seam
	// Cal cumulative Energy
	for y := 1; y < height; y++ {
		for x := 0; x < width; x++ {

			prevRow := (y - 1) * width
			curRow := y * width

			prevRowWithX := prevRow + x

			prevMinOffset := 0
			minValue := cumulativeEnergy[prevRowWithX+prevMinOffset]

			if x > 0 && cumulativeEnergy[prevRowWithX-1] < minValue {
				prevMinOffset = -1
				minValue = cumulativeEnergy[prevRowWithX+prevMinOffset]
			}
			if x < width-1 && cumulativeEnergy[prevRowWithX+1] < minValue {
				prevMinOffset = 1
				minValue = cumulativeEnergy[prevRowWithX+prevMinOffset]
			}

			cumulativeEnergy[curRow+x] += minValue
			prevMinIndex[curRow+x] = int8(prevMinOffset)
		}
	}

	// Cal least energy
	minXValue := cumulativeEnergy[(height-1)*width]
	curX := 0
	for x := 1; x < width; x++ {
		if cumulativeEnergy[(height-1)*width+x] < minXValue {
			minXValue = cumulativeEnergy[(height-1)*width+x]
			curX = x
		}
	}

	// Calculate Seam
	for y := height - 1; y >= 0; y-- {
		seam[y] = curX
		if y > 0 {
			curX += int(prevMinIndex[y*width+curX])
		}
	}

	// Remove Seam
	for y := 0; y < height; y++ {

		curRow := y * image.Stride

		startPos := curRow + int(seam[y])*4
		endPos := curRow + width*4
		copy(image.Data[startPos:endPos], image.Data[startPos+4:endPos])

	}

	image.Width--

}

func main() {

	image, err := LoadImage("test.jpg")
	if err != nil {
		fmt.Println(err.Error())
	}

	energyMap := make([]int, image.Width*image.Height)
	cumulativeEnergy := make([]int, len(energyMap))
	prevMinIndex := make([]int8, len(energyMap))
	seam := make([]int, image.Height)

	for i := 0; i < 50; i++ {
		fmt.Println(i)

		CalculateEnergy(image, energyMap)
		CalculateAndRemoveSeam(image, energyMap, cumulativeEnergy, seam, prevMinIndex)
	}

	image.toImageFile("output.jpg")
}
