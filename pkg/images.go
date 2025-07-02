package barertc

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"git.kirsle.net/apps/barertc/pkg/config"
	"git.kirsle.net/apps/barertc/pkg/log"
	"github.com/edwvee/exiffix"
	"golang.org/x/image/draw"
)

var (
	// TODO: configurable
	MaxPhotoWidth = 1280
)

// ProcessImage treats user uploaded images:
//
// - Scales them down to a reasonable size
// - Strips EXIF metadata
//
// and returns the modified image again as bytes.
//
// Also returns the suggested preview width, height to draw the image
// at. This may be smaller than its true width x height.
//
// Filetype should be image/jpeg, image/gif or image/png.
func ProcessImage(fileType string, data []byte) ([]byte, int, int) {
	reader := bytes.NewReader(data)

	// Strip EXIF data.
	origImage, _, err := exiffix.Decode(reader)
	if err != nil {
		log.Error("ProcessImage: exiffix: %s", err)
		return data, config.Current.PreviewImageWidth, config.Current.PreviewImageWidth
	}

	reader.Seek(0, io.SeekStart)
	var width, height = origImage.Bounds().Max.X, origImage.Bounds().Max.Y

	log.Info("ProcessImage: taking a %dx%d image", width, height)

	// Compute what size we should scale the width/height to,
	// and the even smaller preview size for front-end.
	var (
		previewWidth  = config.Current.PreviewImageWidth
		previewHeight = previewWidth
	)
	if width >= height {
		log.Debug("Its width(%d) is >= its height (%d)", width, height)
		if width > config.Current.MaxImageWidth {
			newWidth := config.Current.MaxImageWidth
			log.Debug("\tnewWidth=%d", newWidth)
			log.Debug("\tnewHeight=(%d / %d) * %d", width, height, newWidth)
			height = int((float64(height) / float64(width)) * float64(newWidth))
			width = newWidth
			log.Debug("Its longest is width, scale to %dx%d", width, height)
		}

		// Compute the preview width.
		if width > config.Current.PreviewImageWidth {
			newWidth := config.Current.PreviewImageWidth
			previewHeight = int((float64(height) / float64(width)) * float64(newWidth))
			previewWidth = newWidth
		}
	} else {
		if height > config.Current.MaxImageWidth {
			newHeight := config.Current.MaxImageWidth
			width = int((float64(width) / float64(height)) * float64(newHeight))
			height = newHeight
			log.Debug("Its longest is height, scale to %dx%d", width, height)
		}

		// Compute the preview height.
		if height > config.Current.PreviewImageWidth {
			newHeight := config.Current.PreviewImageWidth
			previewWidth = int((float64(width) / float64(height)) * float64(newHeight))
			previewHeight = newHeight
			log.Debug("Its longest is height, scale to %dx%d", previewWidth, previewHeight)
		}
	}

	// Scale the image.
	scaledImg := Scale(origImage, image.Rect(0, 0, width, height), draw.ApproxBiLinear)

	// Return the new bytes.
	var buf = bytes.NewBuffer([]byte{})
	switch fileType {
	case "image/jpeg":
		jpeg.Encode(buf, scaledImg, &jpeg.Options{
			Quality: 90,
		})
	case "image/gif":
		// Return the original data - we will only break it.
		return data, width, height
	case "image/png":
		png.Encode(buf, scaledImg)
	default:
		return data, config.Current.PreviewImageWidth, config.Current.PreviewImageWidth
	}

	return buf.Bytes(), previewWidth, previewHeight
}

// Scale down an image. Example:
//
// scaled := Scale(src, image.Rect(0, 0, 200, 200), draw.ApproxBiLinear)
func Scale(src image.Image, rect image.Rectangle, scale draw.Scaler) image.Image {
	dst := image.NewRGBA(rect)
	copyRect := image.Rect(
		rect.Min.X,
		rect.Min.Y,
		rect.Min.X+rect.Max.X,
		rect.Min.Y+rect.Max.Y,
	)
	scale.Scale(dst, copyRect, src, src.Bounds(), draw.Over, nil)
	return dst
}
