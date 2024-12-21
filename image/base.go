package image

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"qrcode/image/styles/moduledrawers"
	"qrcode/utils"
	"strings"
)

type BaseImage struct {
	kind            *string
	allowedKinds    []string
	NeedsContext    bool
	NeedsProcessing bool
	NeedsDrawRect   bool
	border          int
	width           int
	boxSize         int
	pixelSize       int
	modules         [][]bool
	img             image.Image
}

type BaseImageWithDrawer struct {
	BaseImage
	DefaultDrawerClass moduledrawers.QRModuleDrawer
	DrawerAliases      map[string]DrawerAlias
	ModuleDrawer       moduledrawers.QRModuleDrawer
	EyeDrawer          moduledrawers.QRModuleDrawer
}

type DrawerAlias struct {
	DrawerClass moduledrawers.QRModuleDrawer
	Args        map[string]interface{}
}

func NewBaseImage(border int, width int, boxSize int, modules [][]bool) *BaseImage {
	pixelSize := (width + border*2) * boxSize
	img := image.NewRGBA(image.Rect(0, 0, pixelSize, pixelSize))
	return &BaseImage{
		border:        border,
		width:         width,
		boxSize:       boxSize,
		pixelSize:     pixelSize,
		modules:       modules,
		img:           img,
		NeedsDrawRect: true,
	}
}

func (bi *BaseImage) DrawRect(row, col int) error {
	return errors.New("DrawRect method not implemented")
}

func (bi *BaseImage) DrawRectContext(row, col int, qr any) error {
	return errors.New("DrawRectContext method not implemented")
}

func (bi *BaseImage) Process() error {
	return errors.New("Process method not implemented")
}

func (bi *BaseImage) Save(stream *image.Image, kind *string) error {
	return errors.New("Save method not implemented")
}

func (bi *BaseImage) PixelBox(row, col int) (image.Point, image.Point) {
	x := (col + bi.border) * bi.boxSize
	y := (row + bi.border) * bi.boxSize
	return image.Point{x, y}, image.Point{x + bi.boxSize - 1, y + bi.boxSize - 1}
}

func (bi *BaseImage) NewImage() image.Image {
	return image.NewRGBA(image.Rect(0, 0, bi.pixelSize, bi.pixelSize))
}

func (bi *BaseImage) InitNewImage() {
	// No-op for now
}

func (bi *BaseImage) GetImage() image.Image {
	return bi.img
}

func (bi *BaseImage) CheckKind(kind *string, transform func(string) string) (string, error) {
	if kind == nil {
		kind = bi.kind
	}
	allowed := len(bi.allowedKinds) == 0 || utils.Contains(bi.allowedKinds, *kind)
	if transform != nil {
		*kind = transform(*kind)
		if !allowed {
			allowed = utils.Contains(bi.allowedKinds, *kind)
		}
	}
	if !allowed {
		return "", errors.New("Cannot set image type to " + *kind)
	}
	return *kind, nil
}

func (bi *BaseImage) IsEye(row, col int) bool {
	return (row < 7 && col < 7) || (row < 7 && bi.width-col < 8) || (bi.width-row < 8 && col < 7)
}

func NewBaseImageWithDrawer(
	border int,
	width int,
	boxSize int,
	modules [][]bool,
	moduleDrawer moduledrawers.QRModuleDrawer,
	eyeDrawer moduledrawers.QRModuleDrawer,
) *BaseImageWithDrawer {
	baseImage := NewBaseImage(border, width, boxSize, modules)
	return &BaseImageWithDrawer{
		BaseImage:          *baseImage,
		ModuleDrawer:       moduleDrawer,
		EyeDrawer:          eyeDrawer,
		DefaultDrawerClass: moduleDrawer,
		DrawerAliases:      make(map[string]DrawerAlias),
	}
}

func (biwd *BaseImageWithDrawer) GetDefaultModuleDrawer() moduledrawers.QRModuleDrawer {
	return biwd.DefaultDrawerClass
}

func (biwd *BaseImageWithDrawer) GetDefaultEyeDrawer() moduledrawers.QRModuleDrawer {
	return biwd.DefaultDrawerClass
}

func (biwd *BaseImageWithDrawer) InitNewImage() {
	biwd.ModuleDrawer.Initialize(biwd)
	biwd.EyeDrawer.Initialize(biwd)
	biwd.BaseImage.InitNewImage()
}

func (biwd *BaseImageWithDrawer) DrawRectContext(row, col int, qr interface{}) {
	boxStart, _ := biwd.PixelBox(row, col)
	drawer := biwd.ModuleDrawer
	if biwd.IsEye(row, col) {
		drawer = biwd.EyeDrawer
	}

	rectangle := moduledrawers.Rectangle{
		X:      boxStart.X,
		Y:      boxStart.Y,
		Width:  biwd.boxSize,
		Height: biwd.boxSize,
	}

	drawer.DrawRect(rectangle, false)
}

type PilImage struct {
	BaseImage
	fillColor color.Color
	idr       *image.RGBA
}

func NewPilImage(border, width, boxSize int, modules [][]bool, kwargs map[string]interface{}) *PilImage {
	img := NewBaseImage(border, width, boxSize, modules)
	pilImg := &PilImage{
		BaseImage: *img,
	}
	pilImg.img = pilImg.newImage(kwargs)
	return pilImg
}

func (p *PilImage) newImage(kwargs map[string]interface{}) *image.RGBA {
	backColor := color.White
	fillColor := color.Black

	if bc, ok := kwargs["back_color"].(string); ok {
		backColor = parseColor(bc).(color.Gray16)
	}
	if fc, ok := kwargs["fill_color"].(string); ok {
		fillColor = parseColor(fc).(color.Gray16)
	}

	if fillColor == color.Black && backColor == color.White {
		fillColor = color.Black
		backColor = color.White
	} else if r, g, b, a := backColor.RGBA(); r == 0 && g == 0 && b == 0 && a == 0 {
		backColor = color.Gray16{Y: 0}
	} else {
		backColor = color.White
	}

	img := image.NewRGBA(image.Rect(0, 0, p.pixelSize, p.pixelSize))
	draw.Draw(img, img.Bounds(), &image.Uniform{backColor}, image.Point{}, draw.Src)
	p.fillColor = fillColor
	p.idr = img
	return img
}

func (p *PilImage) DrawRect(row, col int) {
	box := p.pixelBox(row, col)
	draw.Draw(p.idr, box, &image.Uniform{p.fillColor}, image.Point{}, draw.Src)
}

func (p *PilImage) Save(stream *os.File, format string, kwargs map[string]interface{}) error {
	if format == "" {
		format = *p.kind
	}
	return png.Encode(stream, p.img)
}

func (p *PilImage) pixelBox(row, col int) image.Rectangle {
	x := p.border + col*p.boxSize
	y := p.border + row*p.boxSize
	return image.Rect(x, y, x+p.boxSize, y+p.boxSize)
}

func parseColor(s string) color.Color {
	s = strings.ToLower(s)
	switch s {
	case "black":
		return color.Black
	case "white":
		return color.White
	case "transparent":
		return color.Transparent
	default:
		return color.Black
	}
}
