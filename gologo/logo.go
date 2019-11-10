// logo := gologo.New()
// logo.AddText("\ue705", 32, "#5FB878", "iconfont.ttf")
// logo.AddText("Fly网络", 18, "#5FB878", "未央简体.ttf")
// logo.SetSpacing(10)
// logoImage, err := logo.GetImage()
// if err != nil {
// 	fmt.Println("err:", err.Error())
// } else {
// 	logo.SavePNG(logoImage, "logo.png")
// }
// fmt.Println("ok")

package gologo

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"math"
	"os"
	"strings"

	"github.com/golang/freetype"
)

type Font struct {
	File  string
	Size  float64
	DPI   float64
	Text  string
	Color string
}

// 默认 dpi 为 72
func NewFont(text string, size float64, color string, file string) Font {
	return Font{
		File:  file,
		Size:  size,
		DPI:   72.0,
		Text:  text,
		Color: color,
	}
}

type logo struct {
	fonts           []Font
	spacing         int // 间距
	backgroundColor color.Color
}

func New() *logo {
	return &logo{
		spacing:         5,
		backgroundColor: image.Transparent,
	}
}

func (this *logo) AddText(text string, size float64, color string, file string) *logo {
	return this.AddFont(NewFont(text, size, color, file))
}

func (this *logo) AddFont(font Font) *logo {
	this.fonts = append(this.fonts, font)
	return this
}

func (this *logo) SetSpacing(spacing int) *logo {
	this.spacing = spacing
	return this
}

func (this *logo) SetBackgroundColor(backgroundColor color.Color) *logo {
	this.backgroundColor = backgroundColor
	return this
}

// 默认保存为png格式
func (this *logo) Save(imgPath string) error {
	img, err := this.GetImage()
	if err != nil {
		return err
	}

	return this.SavePNG(img, imgPath)
}

// 保存为png格式
func (this *logo) SavePNG(img image.Image, imgPath string) error {
	outFile, err := os.Create(imgPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	b := bufio.NewWriter(outFile)
	err = png.Encode(b, img)
	if err != nil {
		return err
	}
	return b.Flush()
}

func (this *logo) GetImage() (image.Image, error) {
	if len(this.fonts) == 0 {
		return nil, errors.New("no fonts")
	}
	// } else if len(this.fonts) == 1 {
	// 	return this.getImage(this.fonts[0])
	// }

	var (
		width, height int
		imgs          []image.Image
	)

	for i, font := range this.fonts {
		img, err := this.getImage(font)
		if err != nil {
			return nil, err
		}

		if i == 0 {
			width = img.Bounds().Dx()
		} else {
			width = width + this.spacing + img.Bounds().Dx()
		}
		if img.Bounds().Dy() > height {
			height = img.Bounds().Dy()
		}

		imgs = append(imgs, img)
	}

	//创建新图层
	canvas := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(this.backgroundColor), image.ZP, draw.Src)

	var w, x, y int
	for i, img := range imgs {
		r := img.Bounds()
		imgW := r.Dx()
		imgH := r.Dy()
		if imgH < height {
			y = int(math.Floor(float64((height - imgH) / 2)))
		} else {
			y = 0
		}

		if i == 0 {
			x = -r.Min.X
			w = imgW
		} else {
			x = (w + this.spacing) - r.Min.X
			w = w + this.spacing + imgW
		}

		//image.ZP代表Point结构体，目标的源点，即(0,0)
		//draw.Src 源图像透过遮罩后，替换掉目标图像
		//draw.Over 源图像透过遮罩后，覆盖在目标图像上（类似图层）

		rect := image.Rect(x, -(r.Min.Y - y), w, r.Max.Y)
		draw.Draw(canvas, rect, img, image.ZP, draw.Over)
	}

	return canvas, nil
}

func (this *logo) getImage(font Font) (image.Image, error) {
	fontBytes, err := ioutil.ReadFile(font.File)
	if err != nil {
		// log.Println("读取字体数据出错")
		return nil, err
	}

	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		// log.Println("转换字体样式出错")
		return nil, err
	}

	width := len(font.Text) * int(font.Size)
	height := int(font.Size * 2)

	c, err := ParseHexColor(font.Color)
	if err != nil {
		return nil, err
	}
	fontColor := image.NewUniform(c)

	// 图片背景颜色
	backgroundColor := image.Transparent
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), backgroundColor, image.ZP, draw.Src)

	ft := freetype.NewContext()
	ft.SetDPI(font.DPI)
	ft.SetFont(f)
	ft.SetFontSize(font.Size)
	ft.SetClip(img.Bounds())
	ft.SetDst(img)
	ft.SetSrc(fontColor)
	//ft.SetSrc(fg)

	pt := freetype.Pt(0, int(font.Size))
	_, err = ft.DrawString(font.Text, pt)

	if err != nil {
		// log.Println("向图片写字体出错")
		return nil, err
	}

	rect := imageRectangle(img, backgroundColor)
	subImage := img.SubImage(image.Rect(rect.Max.X, rect.Min.X, rect.Max.Y, rect.Min.Y)).(*image.RGBA)

	return subImage, nil
}

// 将字体颜色(#5FB878)转成 color.RGBA
func ParseHexColor(s string) (c color.RGBA, err error) {
	c.A = 0xff
	switch len(s) {
	case 7:
		_, err = fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	case 4:
		_, err = fmt.Sscanf(s, "#%1x%1x%1x", &c.R, &c.G, &c.B)
		// Double the hex digits:
		c.R *= 17
		c.G *= 17
		c.B *= 17
	default:
		err = fmt.Errorf("invalid length, must be 7 or 4")

	}
	return
}

var errInvalidFormat = errors.New("invalid format")

func ParseHexColorFast(s string) (c color.RGBA, err error) {
	c.A = 0xff

	if s[0] != '#' {
		return c, errInvalidFormat
	}

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		err = errInvalidFormat
		return 0
	}

	switch len(s) {
	case 7:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
	case 4:
		c.R = hexToByte(s[1]) * 17
		c.G = hexToByte(s[2]) * 17
		c.B = hexToByte(s[3]) * 17
	default:
		err = errInvalidFormat
	}
	return
}

// 比较两个颜色是否相同
func isSameColor(c1, c2 color.Color) bool {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()
	if r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2 {
		return true
	}

	return false
}

func imageRectangle(img image.Image, c color.Color) image.Rectangle {
	return image.Rectangle{
		Min: imagePointMin(img, c),
		Max: imagePointMax(img, c),
	}
}

// image 中查找颜色的坐标
func imagePointMin(img image.Image, c color.Color) image.Point {
	var point image.Point

	p := img.Bounds().Size()
	for x := 0; x < p.Y; x++ {
		b := false
		for y := 0; y < p.X; y++ {
			c1 := img.At(y, x)
			if !isSameColor(c1, c) {
				b = true
				point.X = x
				break
			}
		}

		if b {
			break
		}
	}

	for x := p.Y; x > point.X; x-- {
		b := false
		for y := 0; y < p.X; y++ {
			c1 := img.At(y, x)
			if !isSameColor(c1, c) {
				b = true
				point.Y = x + 1
				break
			}
		}

		if b {
			break
		}
	}

	return point
}

func imagePointMax(img image.Image, c color.Color) image.Point {
	var point image.Point

	p := img.Bounds().Size()
	for x := 0; x < p.X; x++ {
		b := false
		for y := 0; y < p.Y; y++ {
			c1 := img.At(x, y)
			if !isSameColor(c1, c) {
				b = true
				point.X = x
				break
			}
		}

		if b {
			break
		}
	}

	for x := p.X; x > point.X; x-- {
		b := false
		for y := 0; y < p.Y; y++ {
			c1 := img.At(x, y)
			if !isSameColor(c1, c) {
				b = true
				point.Y = x + 1
				break
			}
		}

		if b {
			break
		}
	}

	return point
}

func unicodeToString(form string) (to string, err error) {
	bs, err := hex.DecodeString(strings.Replace(form, `\u`, ``, -1))
	if err != nil {
		return
	}

	for i, bl, br, r := 0, len(bs), bytes.NewReader(bs), uint16(0); i < bl; i += 2 {
		binary.Read(br, binary.BigEndian, &r)
		to += string(r)
	}

	return
}
