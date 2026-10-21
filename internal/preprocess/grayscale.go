package preprocess

import "image/color"

func pillowGray(r, g, b uint8) uint8 {
	return uint8((19595*int(r) + 38470*int(g) + 7471*int(b) + 0x8000) >> 16)
}

func nrgbaGray(c color.NRGBA) uint8 {
	return pillowGray(c.R, c.G, c.B)
}
