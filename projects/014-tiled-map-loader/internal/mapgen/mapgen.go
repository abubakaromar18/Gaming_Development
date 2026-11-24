// Package mapgen procedurally generates the demo map and its tilesets in
// Tiled's JSON format, plus the tileset images. It doubles as a reference
// for the format: everything the loader reads, this package writes.
package mapgen

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
)

// File names produced by Generate.
const (
	MapFile        = "demo.tmj"
	TerrainFile    = "terrain.tsj"
	TerrainImage   = "terrain.png"
	MarkersImage   = "markers.png"
	DefaultSeed    = 14
	mapW, mapH     = 40, 30
	terrainFirst   = 1
	markersFirst   = terrainFirst + terrainCount
	terrainCount   = 11
	waterFrames    = 3
	waterFrameMS   = 300
	terrainColumns = 8
)

// Local tile IDs in the terrain tileset.
const (
	tileGrass = iota
	tileTallGrass
	tileDirt
	tileWall
	tileWater // frames tileWater..tileWater+waterFrames-1
	_
	_
	tileBridge
	tileTree
	tileFlower
	tileRock
)

// Local tile IDs in the embedded marker tileset.
const (
	markerChest = iota
	markerSign
)

// The rock's drawn ellipse; its collision ellipse is derived from it.
const rockCX, rockCY, rockRX, rockRY = 6.5, 10.5, 5.5, 4.5

// Layout landmarks, in tiles.
var (
	spawnTile = [2]int{4, 4}
	pondC     = [2]float64{20, 12}
	pondR     = [2]float64{7.5, 4.5}
	ruins     = [4]int{25, 18, 36, 27} // x0, y0, x1, y1 (exclusive)
	chestTile = [2]int{30, 22}
	signTile  = [2]int{6, 10}
	// The path visits these waypoints in order with straight segments.
	pathPoints = [][2]int{spawnTile, {4, 12}, {30, 12}, chestTile}
)

// Generate builds every file of the demo map from seed. Output is a pure
// function of the seed.
func Generate(seed uint64) (map[string][]byte, error) {
	r := &splitMix64{state: seed}
	ground, decor := layout(r)

	terrainPNG, err := paintTerrain(r).png()
	if err != nil {
		return nil, fmt.Errorf("mapgen: terrain image: %w", err)
	}
	markersPNG, err := paintMarkers().png()
	if err != nil {
		return nil, fmt.Errorf("mapgen: markers image: %w", err)
	}
	decorData, err := packBase64Zlib(decor)
	if err != nil {
		return nil, fmt.Errorf("mapgen: decor layer: %w", err)
	}
	mapJSON, err := marshal(demoMap(seed, ground, decorData))
	if err != nil {
		return nil, err
	}
	tsJSON, err := marshal(terrainTileset())
	if err != nil {
		return nil, err
	}
	return map[string][]byte{
		MapFile: mapJSON, TerrainFile: tsJSON,
		TerrainImage: terrainPNG, MarkersImage: markersPNG,
	}, nil
}

func marshal(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return nil, fmt.Errorf("mapgen: encode json: %w", err)
	}
	return append(b, '\n'), nil
}

// layout returns the ground and decoration layers as GIDs.
func layout(r *splitMix64) (ground, decor []uint32) {
	ground = make([]uint32, mapW*mapH)
	decor = make([]uint32, mapW*mapH)
	set := func(layer []uint32, x, y int, local int) { layer[y*mapW+x] = uint32(terrainFirst + local) }
	get := func(x, y int) int { return int(ground[y*mapW+x]) - terrainFirst }

	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			t := tileGrass
			if r.float() < 0.15 {
				t = tileTallGrass
			}
			if x == 0 || y == 0 || x == mapW-1 || y == mapH-1 {
				t = tileWall
			}
			set(ground, x, y, t)
		}
	}

	// Pond: a noisy ellipse, then one cellular-automaton smoothing pass so
	// the shoreline is lumpy rather than speckled.
	water := make([]bool, mapW*mapH)
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			dx, dy := (float64(x)-pondC[0])/pondR[0], (float64(y)-pondC[1])/pondR[1]
			water[y*mapW+x] = dx*dx+dy*dy < 1+(r.float()-0.5)*0.6
		}
	}
	for y := 1; y < mapH-1; y++ {
		for x := 1; x < mapW-1; x++ {
			n := 0
			for oy := -1; oy <= 1; oy++ {
				for ox := -1; ox <= 1; ox++ {
					if water[(y+oy)*mapW+x+ox] {
						n++
					}
				}
			}
			if n >= 5 {
				set(ground, x, y, tileWater)
			}
		}
	}

	// Ruins: a wall outline with a few collapsed bricks (never the corners).
	for y := ruins[1]; y < ruins[3]; y++ {
		for x := ruins[0]; x < ruins[2]; x++ {
			edgeX, edgeY := x == ruins[0] || x == ruins[2]-1, y == ruins[1] || y == ruins[3]-1
			if (edgeX || edgeY) && ((edgeX && edgeY) || r.float() > 0.12) {
				set(ground, x, y, tileWall)
			}
		}
	}

	// The path is carved last so it always connects spawn to the chest:
	// water under it becomes bridge, walls on it become rubble-free dirt.
	onPath := make([]bool, mapW*mapH)
	for i := 1; i < len(pathPoints); i++ {
		a, b := pathPoints[i-1], pathPoints[i]
		x, y := a[0], a[1]
		for {
			onPath[y*mapW+x] = true
			if get(x, y) == tileWater {
				set(ground, x, y, tileBridge)
			} else if get(x, y) != tileBridge {
				set(ground, x, y, tileDirt)
			}
			if x == b[0] && y == b[1] {
				break
			}
			x += sign(b[0] - x)
			if x == b[0] {
				y += sign(b[1] - y)
			}
		}
	}

	for y := 1; y < mapH-1; y++ {
		for x := 1; x < mapW-1; x++ {
			if t := get(x, y); (t != tileGrass && t != tileTallGrass) || onPath[y*mapW+x] || nearLandmark(x, y) {
				continue
			}
			var flags uint32
			if r.float() < 0.5 {
				flags = 0x80000000 // horizontal flip: free visual variety
			}
			switch v := r.float(); {
			case v < 0.05:
				set(decor, x, y, tileTree)
			case v < 0.10:
				set(decor, x, y, tileFlower)
				decor[y*mapW+x] |= flags
			case v < 0.13:
				set(decor, x, y, tileRock)
				decor[y*mapW+x] |= flags
			}
		}
	}
	return ground, decor
}

func nearLandmark(x, y int) bool {
	near := func(p [2]int, d int) bool { return abs(x-p[0]) <= d && abs(y-p[1]) <= d }
	return near(spawnTile, 2) || near(chestTile, 1) || near(signTile, 1)
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

func abs(v int) int { return int(math.Abs(float64(v))) }

func packBase64Zlib(gids []uint32) (string, error) {
	raw := make([]byte, 4*len(gids))
	for i, g := range gids {
		binary.LittleEndian.PutUint32(raw[i*4:], g)
	}
	var buf bytes.Buffer
	zw, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err := zw.Write(raw); err != nil {
		return "", err
	}
	if err := zw.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
