package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"runtime"
	"sync"
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
		Stride: dstImg.Stride,
		Width:  width,
		Height: height,
	}, nil
}

func CalculateEnergy(image *RawImage, energyMap []int) {
	width, height := image.Width, image.Height

	for y := 1; y < height-1; y++ {
		rowIdx := y * image.Stride
		prevRow := (y - 1) * image.Stride
		nextRow := (y + 1) * image.Stride
		mapOff := y * width

		for x := 0; x < width; x++ {

			leftX, rightX := x-1, x+1
			if x == 0 {
				leftX = x
			}
			if x == width-1 {
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
}

func CalculateAndRemoveSeam(image *RawImage, energyMap, seam []int, prevMinIndex []int8) {
	width, height := image.Width, image.Height

	// Calculate Seam
	// Cal cumulative Energy
	for y := 1; y < height; y++ {
		prevRow := (y - 1) * width
		curRow := y * width

		for x := 0; x < width; x++ {

			prevRowWithX := prevRow + x

			prevMinOffset := 0
			minValue := energyMap[prevRowWithX+prevMinOffset]

			if x > 0 && energyMap[prevRowWithX-1] < minValue {
				prevMinOffset = -1
				minValue = energyMap[prevRowWithX+prevMinOffset]
			}
			if x < width-1 && energyMap[prevRowWithX+1] < minValue {
				prevMinOffset = 1
				minValue = energyMap[prevRowWithX+prevMinOffset]
			}

			energyMap[curRow+x] += minValue
			prevMinIndex[curRow+x] = int8(prevMinOffset)
		}
	}

	// Cal least energy
	minXValue := energyMap[(height-1)*width]
	curX := 0
	for x := 1; x < width; x++ {
		if energyMap[(height-1)*width+x] < minXValue {
			minXValue = energyMap[(height-1)*width+x]
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

func RemoveBatchSeams(img *RawImage, batchSize int, energyMap []int, prevIdx []int8) {
	w, h := img.Width, img.Height

	for y := 1; y < h; y++ {
		curr, prev := y*w, (y-1)*w
		for x := 0; x < w; x++ {
			bestX, m := x, energyMap[prev+x]
			if x > 0 && energyMap[prev+x-1] < m {
				bestX, m = x-1, energyMap[prev+x-1]
			}
			if x < w-1 && energyMap[prev+x+1] < m {
				bestX, m = x+1, energyMap[prev+x+1]
			}
			energyMap[curr+x] += m
			prevIdx[curr+x] = int8(bestX - x)
		}
	}

	toDelete := make([]bool, w*h)

	for b := 0; b < batchSize; b++ {
		minX, minVal := -1, 2147483647
		lastRow := (h - 1) * w
		for x := 0; x < w; x++ {
			if !toDelete[lastRow+x] && energyMap[lastRow+x] < minVal {
				minVal = energyMap[lastRow+x]
				minX = x
			}
		}

		if minX == -1 {
			break
		}

		currX := minX
		for y := h - 1; y >= 0; y-- {
			toDelete[y*w+currX] = true
			energyMap[y*w+currX] = 2147483647
			if y > 0 {
				currX += int(prevIdx[y*w+currX])
			}
		}
	}

	for y := 0; y < h; y++ {
		writeIdx := 0
		rowStart := y * img.Stride
		mapStart := y * w
		for x := 0; x < w; x++ {
			if !toDelete[mapStart+x] {
				if writeIdx != x {
					copy(img.Data[rowStart+writeIdx*4:rowStart+writeIdx*4+4], img.Data[rowStart+x*4:rowStart+x*4+4])
				}
				writeIdx++
			}
		}
	}
	img.Width -= batchSize
}

func CalculateEnergyParallel(img *RawImage, energyMap []int) {
	numCPUs := runtime.NumCPU()
	var wg sync.WaitGroup
	chunkSize := img.Height / numCPUs

	for i := 0; i < numCPUs; i++ {
		startY, endY := i*chunkSize, (i+1)*chunkSize
		if i == numCPUs-1 {
			endY = img.Height
		}
		wg.Add(1)
		go func(yStart, yEnd int) {
			defer wg.Done()
			for y := yStart; y < yEnd; y++ {
				rowOff, mapOff := y*img.Stride, y*img.Width
				upRow := (y - 1) * img.Stride
				downRow := (y + 1) * img.Stride
				if y == 0 {
					upRow = rowOff
				}
				if y == img.Height-1 {
					downRow = rowOff
				}

				for x := 0; x < img.Width; x++ {
					l, r := (x-1)*4, (x+1)*4
					if x == 0 {
						l = 0
					}
					if x == img.Width-1 {
						r = x * 4
					}

					// X-Gradient
					dxR := int(img.Data[rowOff+r]) - int(img.Data[rowOff+l])
					dxG := int(img.Data[rowOff+r+1]) - int(img.Data[rowOff+l+1])
					dxB := int(img.Data[rowOff+r+2]) - int(img.Data[rowOff+l+2])

					// Y-Gradient
					dyR := int(img.Data[downRow+x*4]) - int(img.Data[upRow+x*4])
					dyG := int(img.Data[downRow+x*4+1]) - int(img.Data[upRow+x*4+1])
					dyB := int(img.Data[downRow+x*4+2]) - int(img.Data[upRow+x*4+2])

					energyMap[mapOff+x] = (dxR*dxR + dxG*dxG + dxB*dxB) + (dyR*dyR + dyG*dyG + dyB*dyB)
				}
			}
		}(startY, endY)
	}
	wg.Wait()
}

func mainWithoutConcurrency() {
	imgName := "images/img1.jpg"
	image, err := LoadImage(imgName)
	if err != nil {
		fmt.Println(err.Error())
	}

	energyMap := make([]int, image.Width*image.Height)
	prevMinIndex := make([]int8, len(energyMap))
	seam := make([]int, image.Height)

	CalculateEnergy(image, energyMap)

	for i := 0; i < 50; i++ {
		fmt.Println(i)

		CalculateAndRemoveSeam(image, energyMap, seam, prevMinIndex)
	}

	image.toImageFile("out-" + imgName)
}

func mainWithConcurrency() {
	imgName := "images/img2.jpg"
	img, err := LoadImage(imgName)
	if err != nil {
		fmt.Println("Error loading:", err)
		return
	}

	totalToRemove := 2500
	batchSize := 100

	energyBuf := make([]int, img.Width*img.Height)
	prevIndexBuf := make([]int8, img.Width*img.Height)

	fmt.Printf("Resizing from %d to %d...\n", img.Width, img.Width-totalToRemove)

	for i := 0; i < totalToRemove; i += batchSize {
		CalculateEnergyParallel(img, energyBuf[:img.Width*img.Height])
		RemoveBatchSeams(img, batchSize, energyBuf[:img.Width*img.Height], prevIndexBuf[:img.Width*img.Height])
		fmt.Printf("Removed %d/%d seams\n", i+batchSize, totalToRemove)
	}

	img.toImageFile("out-" + imgName)
}

func main() {
	// Run Seam Carving without Concurrency
	mainWithConcurrency()

	// Run Seam Carving with Concurrency
	mainWithoutConcurrency()
}
