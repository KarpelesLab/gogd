package gogd

// GDInfo returns a map describing the library's capabilities, mirroring
// the shape of PHP's gd_info().
func GDInfo() map[string]any {
	return map[string]any{
		"GD Version":                       "gogd dev",
		"FreeType Support":                 false,
		"FreeType Linkage":                 "",
		"GIF Read Support":                 false,
		"GIF Create Support":               false,
		"JPEG Support":                     false,
		"PNG Support":                      false,
		"WBMP Support":                     false,
		"XPM Support":                      false,
		"XBM Support":                      false,
		"WebP Support":                     false,
		"AVIF Support":                     false,
		"BMP Support":                      false,
		"TGA Read Support":                 false,
		"JIS-mapped Japanese Font Support": false,
	}
}

// ImageTypes returns the bitfield of image formats gogd can read or write.
// Bits will be set as format support lands in later milestones.
func ImageTypes() int {
	return 0
}
