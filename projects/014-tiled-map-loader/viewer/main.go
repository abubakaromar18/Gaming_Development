// Command viewer opens a Tiled map in an Ebitengine window and lets you walk
// around it. All map parsing and gameplay live in the core module; this
// package only reads the keyboard and draws.
//
//	go mod tidy && go run . -map ../maps/demo.tmj
package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	mapPath := flag.String("map", filepath.Join("..", "maps", "demo.tmj"), "Tiled .tmj map to open")
	flag.Parse()

	g, err := newGame(*mapPath)
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowSize(screenW*3, screenH*3)
	ebiten.SetWindowTitle(g.title)
	// Returning ebiten.Termination from Update makes RunGame return nil.
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
