package main

import (
	"archive/zip"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	_ "golang.org/x/image/webp"
)

var (
	screenWidth  = 450
	screenHeight = 800
	scrollSpeed  = 20.0
)

func clampi(v, left, right int) int {
	if v < left {
		return left
	}
	if v > right {
		return right
	}
	return v
}

type Game struct {
	curentPosY   float64
	currentPage  int
	webtoon      bool
	canvasHeight float64
	pages        []*ebiten.Image
}

func (g *Game) scrollY(dy float64) {
	g.curentPosY += dy
	if g.curentPosY > 0 {
		g.curentPosY = 0
	}
	if g.canvasHeight-float64(screenHeight) < -g.curentPosY {
		g.curentPosY = -(g.canvasHeight - float64(screenHeight))
	}
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.currentPage = clampi(g.currentPage-1, 0, len(g.pages)-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.currentPage = clampi(g.currentPage+1, 0, len(g.pages)-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		g.webtoon = !g.webtoon
	}

	if g.webtoon && ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.scrollY(scrollSpeed)
	}
	if g.webtoon && ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.scrollY(-scrollSpeed)
	}
	if g.webtoon {
		_, dy := ebiten.Wheel()
		g.scrollY(dy * scrollSpeed * 2)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		page := g.pages[g.currentPage]
		if page != nil {
			size := page.Bounds().Size()
			ebiten.SetWindowSize(size.X, size.Y)
		}
	}

	return nil
}

func (g *Game) DrawWebtoon(screen *ebiten.Image) {
	y := 0.0
	for _, page := range g.pages {
		if page == nil {
			continue
		}
		size := page.Bounds().Size()
		scale := float64(screenWidth) / float64(size.X) // by width
		posY := g.curentPosY + y
		y += float64(size.Y) * scale
		if posY < -float64(size.Y)*scale || posY > float64(screenHeight) {
			continue
		}
		opt := ebiten.DrawImageOptions{}
		opt.GeoM.Scale(scale, scale)
		opt.GeoM.Translate(0, posY)
		opt.Filter = ebiten.FilterLinear
		screen.DrawImage(page, &opt)
	}
	g.canvasHeight = y
}

func (g *Game) DrawManga(screen *ebiten.Image) {
	page := g.pages[g.currentPage]
	if page == nil {
		return
	}
	size := page.Bounds().Size()

	// fit screen
	sw := float64(screenWidth) / float64(size.X)
	sh := float64(screenHeight) / float64(size.Y)
	scale := math.Min(sw, sh)

	opt := ebiten.DrawImageOptions{}
	opt.GeoM.Scale(scale, scale)
	opt.Filter = ebiten.FilterLinear
	screen.DrawImage(page, &opt)
}

func (g *Game) Draw(screen *ebiten.Image) {
	if len(g.pages) == 0 {
		return
	}

	if g.webtoon {
		g.DrawWebtoon(screen)
	} else {
		g.DrawManga(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	screenWidth = outsideWidth
	screenHeight = outsideHeight
	return screenWidth, screenHeight
}

func isImageFormat(name string) bool {
	return strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") || strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".webp")
}

func NewGame(filepath string) *Game {
	r, err := zip.OpenReader(filepath)
	if err != nil {
		log.Fatalf("[ERR] Failed top open %s: %v", filepath, err)
	}
	defer r.Close()

	n := 0
	for _, f := range r.File {
		if !f.FileInfo().IsDir() && isImageFormat(f.Name) {
			n += 1
		}
	}
	game := Game{
		currentPage: 0,
		webtoon:     false,
		curentPosY:  0,
		pages:       make([]*ebiten.Image, n),
	}

	go func() {
		r, err := zip.OpenReader(filepath)
		if err != nil {
			log.Fatalf("[ERR] Failed top open %s: %v", filepath, err)
		}
		defer r.Close()

		i := 0
		for _, f := range r.File {
			if f.FileInfo().IsDir() {
				continue
			}
			log.Printf("[DBG] [%d/%d] file: %s", i+1, n, f.Name)
			rc, err := f.Open()
			if err != nil {
				log.Printf("[ERR] failed to open entry %s: %v", f.Name, err)
				continue
			}
			defer rc.Close()
			img, format, err := image.Decode(rc)
			if err != nil {
				log.Printf("[ERR] failed to read image %s of format %s: %v", f.Name, format, err)
				continue
			}
			game.pages[i] = ebiten.NewImageFromImage(img)
			i += 1
		}

		log.Printf("[DBG] loaded %d pages", len(game.pages))
	}()

	return &game
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Requires path to a file")
	}
	filepath := os.Args[1]

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("CBZ Viewer")
	if err := ebiten.RunGame(NewGame(filepath)); err != nil {
		log.Fatal(err)
	}
}
