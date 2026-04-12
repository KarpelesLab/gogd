package gogd

import (
	"bytes"
	"image"
	"io"
	"os"
)

// Image type codes, matching PHP's IMAGETYPE_* constants.
const (
	ImageTypeUnknown = 0
	ImageTypeGIF     = 1
	ImageTypeJPEG    = 2
	ImageTypePNG     = 3
	ImageTypeSWF     = 4
	ImageTypePSD     = 5
	ImageTypeBMP     = 6
	ImageTypeTIFFII  = 7
	ImageTypeTIFFMM  = 8
	ImageTypeJPC     = 9
	ImageTypeJP2     = 10
	ImageTypeJPX     = 11
	ImageTypeJB2     = 12
	ImageTypeSWC     = 13
	ImageTypeIFF     = 14
	ImageTypeWBMP    = 15
	ImageTypeXBM     = 16
	ImageTypeICO     = 17
	ImageTypeWEBP    = 18
	ImageTypeAVIF    = 19
)

// Format bitmask constants, matching PHP's IMG_* constants used by
// ImageTypes() and a handful of gd functions.
const (
	ImgGIF  = 1
	ImgJPEG = 2
	ImgPNG  = 4
	ImgWBMP = 8
	ImgXPM  = 16
	ImgWEBP = 32
	ImgBMP  = 64
	ImgTGA  = 128
	ImgAVIF = 256
)

// ImageTypeToExtension returns the conventional filename extension for a
// gd image type code. When includeDot is true the leading "." is included.
// Returns the empty string for unknown types.
func ImageTypeToExtension(typ int, includeDot bool) string {
	var ext string
	switch typ {
	case ImageTypeGIF:
		ext = "gif"
	case ImageTypeJPEG:
		ext = "jpeg"
	case ImageTypePNG:
		ext = "png"
	case ImageTypeBMP:
		ext = "bmp"
	case ImageTypeWEBP:
		ext = "webp"
	case ImageTypeAVIF:
		ext = "avif"
	case ImageTypeTIFFII, ImageTypeTIFFMM:
		ext = "tiff"
	case ImageTypeWBMP:
		ext = "wbmp"
	case ImageTypeXBM:
		ext = "xbm"
	case ImageTypeICO:
		ext = "ico"
	case ImageTypeSWF, ImageTypeSWC:
		ext = "swf"
	case ImageTypePSD:
		ext = "psd"
	case ImageTypeJPC:
		ext = "jpc"
	case ImageTypeJP2:
		ext = "jp2"
	case ImageTypeJPX:
		ext = "jpx"
	case ImageTypeJB2:
		ext = "jb2"
	case ImageTypeIFF:
		ext = "iff"
	default:
		return ""
	}
	if includeDot {
		return "." + ext
	}
	return ext
}

// ImageTypeToMimeType returns the MIME type for a gd image type code.
func ImageTypeToMimeType(typ int) string {
	switch typ {
	case ImageTypeGIF:
		return "image/gif"
	case ImageTypeJPEG:
		return "image/jpeg"
	case ImageTypePNG:
		return "image/png"
	case ImageTypeBMP:
		return "image/bmp"
	case ImageTypeWEBP:
		return "image/webp"
	case ImageTypeAVIF:
		return "image/avif"
	case ImageTypeTIFFII, ImageTypeTIFFMM:
		return "image/tiff"
	case ImageTypeWBMP:
		return "image/vnd.wap.wbmp"
	case ImageTypeXBM:
		return "image/x-xbitmap"
	case ImageTypeICO:
		return "image/x-icon"
	case ImageTypeSWF, ImageTypeSWC:
		return "application/x-shockwave-flash"
	case ImageTypePSD:
		return "image/vnd.adobe.photoshop"
	case ImageTypeIFF:
		return "image/iff"
	case ImageTypeJPC, ImageTypeJP2, ImageTypeJPX, ImageTypeJB2:
		return "image/jp2"
	}
	return "application/octet-stream"
}

// ImageSize is the dimension and type info returned by [GetImageSize] and
// [GetImageSizeFromString].
type ImageSize struct {
	Width    int
	Height   int
	Type     int    // IMAGETYPE_*
	MimeType string // "image/png", etc.
}

// GetImageSize returns the width, height, format, and MIME type of the
// image file at path without fully decoding its pixels.
func GetImageSize(path string) (*ImageSize, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return imageSize(f)
}

// GetImageSizeFromString is like [GetImageSize] but reads the image from
// an in-memory byte slice.
func GetImageSizeFromString(data []byte) (*ImageSize, error) {
	return imageSize(bytes.NewReader(data))
}

func imageSize(r io.Reader) (*ImageSize, error) {
	cfg, format, err := image.DecodeConfig(r)
	if err != nil {
		return nil, err
	}
	typ := formatToImageType(format)
	return &ImageSize{
		Width:    cfg.Width,
		Height:   cfg.Height,
		Type:     typ,
		MimeType: ImageTypeToMimeType(typ),
	}, nil
}

func formatToImageType(format string) int {
	switch format {
	case "png":
		return ImageTypePNG
	case "jpeg":
		return ImageTypeJPEG
	case "gif":
		return ImageTypeGIF
	case "bmp":
		return ImageTypeBMP
	case "webp":
		return ImageTypeWEBP
	}
	return ImageTypeUnknown
}
