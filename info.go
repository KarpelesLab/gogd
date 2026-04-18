package gogd

// GDInfo returns a map describing the library's capabilities, mirroring
// the shape of PHP's gd_info().
func GDInfo() map[string]any {
	return map[string]any{
		"GD Version":                       "gogd dev",
		"FreeType Support":                 false,
		"FreeType Linkage":                 "",
		"GIF Read Support":                 true,
		"GIF Create Support":               true,
		"JPEG Support":                     true,
		"PNG Support":                      true,
		"WBMP Support":                     false,
		"XPM Support":                      false,
		"XBM Support":                      false,
		"WebP Support":                     true,
		"AVIF Support":                     true,
		"BMP Support":                      true,
		"TGA Read Support":                 false,
		"JIS-mapped Japanese Font Support": false,
	}
}

// ImageTypes returns the bitfield of image formats gogd can read or
// write. Matches PHP's imagetypes() return value.
func ImageTypes() int {
	return ImgGIF | ImgJPEG | ImgPNG | ImgBMP | ImgWEBP | ImgAVIF
}
