package main

import (
	"bitmap-renderer/utils"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/image/bmp"
)

type encBitmap struct {
	width, height int
	byteArray     []byte
}

type gmemDirection int

const (
	MEMDIR_VERTICAL   gmemDirection = iota
	MEMDIR_HORIZONTAL gmemDirection = iota
)

func printHelp() {
	fmt.Printf("\n  Bitmap renderer usage:\n\n\tbitmap-renderer [arguments] -o <output path> <path to image>\n\n")
	fmt.Printf("\tIt will produce the <output path>_bmp.h file with\n")
	fmt.Printf("\tbitmaps and structures describing the bitmap and the\n")
	fmt.Printf("\t<output path>_preview.bmp image if preview is enabled.\n\n")

	fmt.Printf("  Bitmap renderer arguments:\n\n")
	fmt.Printf("\t-o, --output         path to output files (without file extension)\n\n")

	fmt.Printf("\t-d, --color-depth    bits per pixel (default: 4)\n")
	fmt.Printf("\t-mb,--mem-block      bits per graphic memory block (default: 8)\n")
	fmt.Printf("\t-md,--mem-dir        pixels direction in graphic memory (values: vertical, horizontal)\n\n")

	fmt.Printf("\t-i, --invert         invert the image\n\n")
}

func main() {
	var out, source string
	var colorDepth int = 4
	var gmemDir gmemDirection = MEMDIR_HORIZONTAL
	var gmemBlockSize int = 8
	var invert bool

	args := os.Args
	if len(args) <= 1 {
		printHelp()
		os.Exit(0)
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--help":
			printHelp()
			os.Exit(0)
		case "-d", "--color-depth":
			d, err := strconv.ParseUint(args[i+1], 10, 32)
			if err != nil {
				panic(err)
			}
			colorDepth = int(d)
			i++
		case "-o", "--output":
			out = args[i+1]
			i++
		case "-mb", "--mem-block":
			b, err := strconv.ParseUint(args[i+1], 10, 32)
			if err != nil {
				panic(err)
			}
			gmemBlockSize = int(b)
			i++
		case "-md", "--mem-dir":
			dir := args[i+1]
			switch dir {
			case "hor", "horizontal":
				gmemDir = MEMDIR_HORIZONTAL
			default:
				gmemDir = MEMDIR_VERTICAL
			}
			i++
		case "-i", "--invert":
			invert = true
		default:
			source = arg
		}
	}

	if out == "" {
		panic("Missing output")
	}
	if source == "" {
		panic("Missing font")
	}

	bitmap := loadImage(source)
	encoded := encodeBitmap(bitmap, colorDepth, gmemBlockSize, gmemDir, invert)
	saveBitmap(encoded, out)
}

func loadImage(path string) (img image.Image) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".png":
		img, err = png.Decode(f)
		if err != nil {
			panic(err)
		}
	case ".bmp":
		img, err = bmp.Decode(f)
		if err != nil {
			panic(err)
		}
	default:
		fmt.Println("Unknown file extension:", ext)
		os.Exit(22)
	}

	return
}

func encodeBitmap(img image.Image, depth, gmemBlockSize int, gmemDir gmemDirection, invert bool) (bmp encBitmap) {
	bmp.width = img.Bounds().Max.X
	bmp.height = img.Bounds().Max.Y

	pxPerBlock := (gmemBlockSize + depth - 1) / depth
	pxBlockMask := uint32(1<<depth) - 1
	bytePerBlock := (gmemBlockSize + 7) / 8
	bmpByteIdx := 0

	switch gmemDir {
	case MEMDIR_VERTICAL:
		for px := 0; px < bmp.width; px++ {
			for yblock := 0; yblock < (bmp.height+pxPerBlock-1)/pxPerBlock; yblock++ {
				var memblock uint64
				for blockpy := 0; blockpy < pxPerBlock; blockpy++ {
					py := yblock*pxPerBlock + blockpy
					color := img.At(px, py)
					r, g, b, _ := color.RGBA()
					c := (r + g + b) / 3
					if invert {
						c = 0xFFFF - c
					}
					memblock |= uint64(utils.Remap(c, 0, 0xFFFF, 0, pxBlockMask)&pxBlockMask) << (depth - blockpy*depth)
				}
				block := utils.ToByteArray(memblock)
				for b := 0; b < bytePerBlock; b++ {
					bmp.byteArray[bmpByteIdx] = block[b]
					bmpByteIdx++
				}
			}
		}
	case MEMDIR_HORIZONTAL:
		for py := 0; py < bmp.height; py++ {
			for xblock := 0; xblock < (bmp.width+pxPerBlock-1)/pxPerBlock; xblock++ {
				var memblock uint64
				for blockpx := 0; blockpx < pxPerBlock; blockpx++ {
					px := xblock*pxPerBlock + blockpx
					color := img.At(px, py)
					r, g, b, _ := color.RGBA()
					c := (r + g + b) / 3
					if invert {
						c = 0xFFFF - c
					}
					memblock |= uint64(utils.Remap(c, 0, 0xFFFF, 0, pxBlockMask)&pxBlockMask) << (depth - blockpx*depth)
				}
				block := utils.ToByteArray(memblock)
				for b := 0; b < bytePerBlock; b++ {
					bmp.byteArray[bmpByteIdx] = block[b]
					bmpByteIdx++
				}
			}
		}
	}

	return
}

func saveBitmap(bmp encBitmap, dest string) {
	dest = strings.ReplaceAll(dest, "'", "")
	dest = strings.ReplaceAll(dest, "\"", "")
	dest = strings.ReplaceAll(dest, "\\", "/")

	bmpNameIdx := strings.LastIndex(dest, "/") + 1
	bmpName := dest[bmpNameIdx:]
	bmpName = strings.ReplaceAll(bmpName, " ", "_")
	bmpName = strings.ReplaceAll(bmpName, "-", "_")

	outFile, err := os.Create(dest + "_font.h")
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	var (
		strBmpDesc   string
		strBmpStruct string
		strBitmap    string
		strBmp       string
	)

	var (
		bitmapsWeight     int = len(bmp.byteArray)
		descriptorsWeight int = 6
	)

	strBmpStruct = `typedef struct {
	const unsigned char *map;
	unsigned char w;
	unsigned char h;
} __bitmap_t;

`

	strBmp = fmt.Sprintf(`const __bitmap_t %s_font = {
	.map = __%s_map,
	.w = %d,
	.h = %d,
};

`, bmpName, bmpName, bmp.width, bmp.height)

	strBitmap = fmt.Sprintf("const unsigned char __%s_map = {\n", bmpName)
	for i, b := range bmp.byteArray {
		if i%32 == 0 {
			strBitmap += "\t"
		}
		strBitmap += fmt.Sprintf("0x%02X,", b)
		if i%32 == 31 {
			strBitmap += "\n"
		}
	}
	strBitmap += "};\n\n"

	strBmpDesc = fmt.Sprintf("//\t%s bitmap\n//\n//\tMemory usage\n//\t\tBitmap: %d\n//\t\tDescriptor: %d\n//\n//\t\tTotal: %d\n//\n\n",
		bmpName, bitmapsWeight, descriptorsWeight, bitmapsWeight+descriptorsWeight)

	outFile.Write([]byte(fmt.Sprintf("#ifndef __BITMAP_%s_H\n", strings.ToUpper(bmpName))))
	outFile.Write([]byte(fmt.Sprintf("#define __BITMAP_%s_H\n\n", strings.ToUpper(bmpName))))
	outFile.Write([]byte(strBmpDesc))
	outFile.Write([]byte(strBmpStruct))
	outFile.Write([]byte(strBitmap))
	outFile.Write([]byte(strBmp))
	outFile.Write([]byte(fmt.Sprintf("#endif // __BITMAP_%s_H\n", strings.ToUpper(bmpName))))
}
