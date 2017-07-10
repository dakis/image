package image

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/rwcarlsen/goexif/exif"
	"gopkg.in/gographics/imagick.v2/imagick"
	"math"
	"strconv"
)

const (
	IMAGE_INVERT_VERTICAL   = "v"
	IMAGE_INVERT_HORIZONTAL = "h"
)

type Image struct {
	wand         *imagick.MagickWand
	quantumRange uint
	orientation  imagick.OrientationType
}

func New(data *bytes.Buffer) *Image {
	if data == nil {
		return nil
	}

	i := &Image{
		wand: imagick.NewMagickWand(),
	}

	if err := i.SetBytes(data.Bytes()); err != nil {
		return nil
	}

	exifData, err := exif.Decode(bytes.NewReader(data.Bytes()))

	if err == nil {
		tag, err := exifData.Get(exif.Orientation)

		if err == nil {
			orientation, err := strconv.ParseInt(tag.String(), 10, 0)

			if err == nil {
				i.orientation = imagick.OrientationType(orientation)
			}
		}
	}

	if err != nil {
		i.orientation = i.wand.GetImageOrientation()
	}

	return i
}

func (i *Image) Destroy() {
	i.wand.Destroy()
}

func (i *Image) Width() uint {
	return i.wand.GetImageWidth()
}

func (i *Image) Height() uint {
	return i.wand.GetImageHeight()
}

func (i *Image) Format() string {
	return i.wand.GetImageFormat()
}

func (i *Image) QuantumRange() uint {
	if i.quantumRange == 0 {
		_, i.quantumRange = imagick.GetQuantumRange()
	}

	return i.quantumRange
}

func (i *Image) SetProperty(key, value string) error {
	return i.wand.SetImageProperty(key, value)
}

func (i *Image) Property(property string) string {
	return i.wand.GetImageProperty(property)
}

func (i *Image) SetProfile(name string, value []byte) error {
	return i.wand.ProfileImage(name, value)
}

func (i *Image) Orientate() (err error) {
	pw := imagick.NewPixelWand()
	defer pw.Destroy()

	pw.SetColor("none")

	switch i.orientation {
	case imagick.ORIENTATION_TOP_LEFT:
	case imagick.ORIENTATION_TOP_RIGHT:
		i.wand.FlopImage()
	case imagick.ORIENTATION_BOTTOM_RIGHT:
		i.wand.RotateImage(pw, 180)
	case imagick.ORIENTATION_BOTTOM_LEFT:
		i.wand.FlipImage()
	case imagick.ORIENTATION_LEFT_TOP:
		i.wand.FlipImage()
		i.wand.RotateImage(pw, 90)
	case imagick.ORIENTATION_RIGHT_TOP:
		i.wand.RotateImage(pw, 90)
	case imagick.ORIENTATION_RIGHT_BOTTOM:
		i.wand.FlopImage()
		i.wand.RotateImage(pw, 90)
	case imagick.ORIENTATION_LEFT_BOTTOM:
		i.wand.RotateImage(pw, -90)
	case imagick.ORIENTATION_UNDEFINED:
		return fmt.Errorf("no orientation data found in file")
	}

	i.SetOrientation(imagick.ORIENTATION_TOP_LEFT)

	return
}

func (i *Image) SetOrientation(orientation imagick.OrientationType) {
	i.orientation = orientation
	i.wand.SetImageOrientation(orientation)
	i.wand.SetImageProperty("exif:orientation", string(orientation))
}

func (i *Image) Rotate(angle float64) (err error) {
	if angle > 359 {
		angle = math.Mod(angle, 360)
	}

	if angle > 0 {
		pw := imagick.NewPixelWand()
		defer pw.Destroy()

		pw.SetColor("none")

		err = i.wand.RotateImage(pw, angle)
	}

	return
}

func (i *Image) normalizeDimensions(targetWidth, targetHeight uint) (uint, uint) {
	var ratioWidth, ratioHeight float64

	if targetWidth != 0 {
		ratioWidth = float64(i.Width()) / float64(targetWidth)
	}

	if targetHeight != 0 {
		ratioHeight = float64(i.Height()) / float64(targetHeight)
	}

	ratio := math.Max(ratioWidth, ratioHeight)

	return uint(float64(i.Width())/ratio + 0.5), uint(float64(i.Height())/ratio + 0.5)
}

func (i *Image) Resize(targetWidth, targetHeight uint, aspectRatio bool) (err error) {
	if aspectRatio {
		targetWidth, targetHeight = i.normalizeDimensions(targetWidth, targetHeight)
	} else {
		if targetWidth == 0 {
			targetWidth = i.Width()
		}

		if targetHeight == 0 {
			targetHeight = i.Height()
		}
	}

	if targetWidth == i.Width() && targetHeight == i.Height() {
		return nil
	}

	return i.wand.ResizeImage(targetWidth, targetHeight, imagick.FILTER_LANCZOS2, 1)
}

func (i *Image) Extend(width, height uint) (err error) {
	if width == 0 && height == i.Height() ||
		height == 0 && width == i.Width() ||
		width == i.Width() && height == i.Height() {
		return nil
	}

	x := int(i.Width()/2 - width/2)
	y := int(i.Height()/2 - height/2)

	return i.wand.ExtentImage(width, height, x, y)
}

func (i *Image) Crop(x, y int, width, height uint) (err error) {
	return i.wand.CropImage(width, height, x, y)
}

func (i *Image) Invert(direction string) (err error) {
	switch direction {
	case IMAGE_INVERT_VERTICAL:
		i.wand.FlipImage()
	case IMAGE_INVERT_HORIZONTAL:
		i.wand.FlopImage()
	default:
		err = errors.New("wrong direction")
	}

	return
}

func (i *Image) SetBrightness(brightness float64) (err error) {
	return i.wand.BrightnessContrastImage(brightness, 0)
}

func (i *Image) SetContrast(contrast float64) (err error) {
	return i.wand.BrightnessContrastImage(0, contrast)
}

func (i *Image) SetGrayscale() (err error) {
	return i.wand.SetImageType(imagick.IMAGE_TYPE_GRAYSCALE)
}

func (i *Image) SetWhiteFade(targetWhiteFade string) (err error) {
	colorize := imagick.NewPixelWand()
	opacity := imagick.NewPixelWand()
	var color string = fmt.Sprintf("hsl(0, 0%%, %s%%)", targetWhiteFade)

	defer colorize.Destroy()
	defer opacity.Destroy()

	colorize.SetColor("white")
	opacity.SetColor(color)
	return i.wand.ColorizeImage(colorize, opacity)
}

func (i *Image) SetSepia(threshold float64) (err error) {
	return i.wand.SepiaToneImage((threshold * float64(i.QuantumRange())) / 100.0)
}

func (i *Image) AddOverlay(overlay *Image, x, y int) (err error) {
	return i.wand.CompositeImage(overlay.wand, imagick.COMPOSITE_OP_OVER, x, y)
}

func (i *Image) Optimize() error {
	switch format := i.Format(); format {
	case "JPEG":
		i.optimizeJpeg()
	case "PNG":
		i.optimizePng()
	}

	return i.wand.StripImage()
}

func (i *Image) Strip() error {
	return i.wand.StripImage()
}

func (i *Image) SetFormat(format string) (err error) {
	return i.wand.SetImageFormat(format)
}

func (i *Image) FixRedEye(x, y int, width, height uint, color string) (err error) {
	if width == 0 || height == 0 {
		return fmt.Errorf("region cannot be null red eye reduction")
	}

	zone := i.wand.NewPixelRegionIterator(x, y, width, height)
	defer zone.Destroy()

	if zone == nil {
		return fmt.Errorf("zone cannot be loaded for red eye modification")
	}

	for y := 0; y < int(height); y++ {
		row := zone.GetNextIteratorRow()

		for _, pixel := range row {
			red := pixel.GetRed()
			green := pixel.GetGreen()
			blue := pixel.GetBlue()

			redIntensity := red / ((green + blue) / 2)

			if redIntensity > 1.5 {
				if color == "auto" {
					color = fmt.Sprintf("rgb(%0.0f,%0.0f,%0.0f)", 255*(green+blue)/2, 255*green, 255*blue)
				}

				if !pixel.SetColor(color) {
					return fmt.Errorf("red eye: could not set pixel color to %s", color)
				}
			}
		}

		if err := zone.SyncIterator(); err != nil {
			return err
		}
	}

	return
}

func (i *Image) Bytes() []byte {
	return i.wand.GetImageBlob()
}

func (i *Image) SetBytes(data []byte) error {
	return i.wand.ReadImageBlob(data)
}

func (i *Image) optimizePng() {
}

func (i *Image) optimizeJpeg() {
	i.wand.SetInterlaceScheme(imagick.INTERLACE_PLANE)
	i.wand.SetImageType(imagick.IMAGE_TYPE_OPTIMIZE)
	i.wand.SetImageFormat("PJPEG")
	i.wand.SetImageCompressionQuality(60)
	i.wand.SetImageProperty("jpeg:optimize-coding", "true")
	i.wand.SetImageProperty("jpeg:dct-method", "float")
}
