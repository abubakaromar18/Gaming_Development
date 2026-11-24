package mapgen

// Writer-side mirrors of Tiled's JSON format (version 1.10). Field order
// follows what Tiled itself writes, which keeps diffs against
// editor-exported files readable.

type prop struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

type object struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Type       string  `json:"type"` // Tiled 1.10 writes an object's class as "type"
	GID        uint32  `json:"gid,omitempty"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	Rotation   float64 `json:"rotation"`
	Visible    bool    `json:"visible"`
	Point      bool    `json:"point,omitempty"`
	Ellipse    bool    `json:"ellipse,omitempty"`
	Properties []prop  `json:"properties,omitempty"`
}

type layer struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Class       string   `json:"class,omitempty"`
	Visible     bool     `json:"visible"`
	Opacity     float64  `json:"opacity"`
	X           int      `json:"x"`
	Y           int      `json:"y"`
	Width       int      `json:"width,omitempty"`
	Height      int      `json:"height,omitempty"`
	Encoding    string   `json:"encoding,omitempty"`
	Compression string   `json:"compression,omitempty"`
	Data        any      `json:"data,omitempty"`
	DrawOrder   string   `json:"draworder,omitempty"`
	Objects     []object `json:"objects,omitempty"`
	Layers      []layer  `json:"layers,omitempty"`
	Properties  []prop   `json:"properties,omitempty"`
}

type frame struct {
	TileID   int `json:"tileid"`
	Duration int `json:"duration"`
}

type tile struct {
	ID          int     `json:"id"`
	Type        string  `json:"type,omitempty"`
	Properties  []prop  `json:"properties,omitempty"`
	ObjectGroup *layer  `json:"objectgroup,omitempty"`
	Animation   []frame `json:"animation,omitempty"`
}

type tileset struct {
	FirstGID    int    `json:"firstgid,omitempty"`
	Type        string `json:"type,omitempty"`
	Name        string `json:"name"`
	TileWidth   int    `json:"tilewidth"`
	TileHeight  int    `json:"tileheight"`
	Spacing     int    `json:"spacing"`
	Margin      int    `json:"margin"`
	TileCount   int    `json:"tilecount"`
	Columns     int    `json:"columns"`
	Image       string `json:"image"`
	ImageWidth  int    `json:"imagewidth"`
	ImageHeight int    `json:"imageheight"`
	Tiles       []tile `json:"tiles,omitempty"`
}

type tilesetRef struct {
	FirstGID int    `json:"firstgid"`
	Source   string `json:"source"`
}

type tiledMap struct {
	Type            string  `json:"type"`
	Version         string  `json:"version"`
	TiledVersion    string  `json:"tiledversion"`
	Orientation     string  `json:"orientation"`
	RenderOrder     string  `json:"renderorder"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	TileWidth       int     `json:"tilewidth"`
	TileHeight      int     `json:"tileheight"`
	Infinite        bool    `json:"infinite"`
	BackgroundColor string  `json:"backgroundcolor"`
	NextLayerID     int     `json:"nextlayerid"`
	NextObjectID    int     `json:"nextobjectid"`
	Properties      []prop  `json:"properties"`
	Tilesets        []any   `json:"tilesets"`
	Layers          []layer `json:"layers"`
}

func glyph(g string) prop { return prop{"glyph", "string", g} }

func terrainTileset() tileset {
	rx, ry, rw, rh := ellipseBounds(rockCX, rockCY, rockRX, rockRY)
	var water []frame
	for f := 0; f < waterFrames; f++ {
		water = append(water, frame{TileID: tileWater + f, Duration: waterFrameMS})
	}
	return tileset{
		Type: "tileset", Name: "terrain",
		TileWidth: tilePx, TileHeight: tilePx, TileCount: terrainCount, Columns: terrainColumns,
		Image: TerrainImage, ImageWidth: terrainColumns * tilePx, ImageHeight: 2 * tilePx,
		Tiles: []tile{
			{ID: tileGrass, Type: "ground", Properties: []prop{glyph(".")}},
			{ID: tileTallGrass, Type: "ground", Properties: []prop{glyph(",")}},
			{ID: tileDirt, Type: "ground", Properties: []prop{glyph(":")}},
			{ID: tileWall, Type: "wall", Properties: []prop{glyph("#"), {"solid", "bool", true}}},
			{ID: tileWater, Type: "water", Animation: water,
				Properties: []prop{glyph("~"), {"solid", "bool", true}}},
			{ID: tileBridge, Type: "ground", Properties: []prop{glyph("=")}},
			{ID: tileTree, Type: "tree", Properties: []prop{glyph("T")},
				ObjectGroup: &layer{Name: "", Type: "objectgroup", DrawOrder: "index", Visible: true, Opacity: 1,
					Objects: []object{{ID: 1, Name: "trunk", X: 5, Y: 9, Width: 6, Height: 7, Visible: true}}}},
			{ID: tileFlower, Type: "decor", Properties: []prop{glyph("*")}},
			{ID: tileRock, Type: "rock", Properties: []prop{glyph("o")},
				ObjectGroup: &layer{Name: "", Type: "objectgroup", DrawOrder: "index", Visible: true, Opacity: 1,
					Objects: []object{{ID: 1, Name: "rock", X: rx, Y: ry, Width: rw, Height: rh, Visible: true, Ellipse: true}}}},
		},
	}
}

func demoMap(seed uint64, ground []uint32, decorBase64 string) tiledMap {
	px := func(t int) float64 { return float64(t * tilePx) }
	markers := tileset{
		FirstGID: markersFirst, Name: "markers",
		TileWidth: tilePx, TileHeight: tilePx, TileCount: 2, Columns: 2,
		Image: MarkersImage, ImageWidth: 2 * tilePx, ImageHeight: tilePx,
		Tiles: []tile{
			{ID: markerChest, Type: "chest", Properties: []prop{glyph("$")}},
			{ID: markerSign, Type: "sign", Properties: []prop{glyph("?")}},
		},
	}
	bridgeY := pathPoints[1][1]
	entities := []object{
		{ID: 1, Name: "player_start", Type: "spawn", X: px(spawnTile[0]) + 8, Y: px(spawnTile[1]) + 8, Visible: true, Point: true},
		// Tile objects are anchored at their bottom-left corner.
		{ID: 2, Name: "lantern_chest", Type: "chest", GID: markersFirst + markerChest,
			X: px(chestTile[0]), Y: px(chestTile[1] + 1), Width: tilePx, Height: tilePx, Visible: true,
			Properties: []prop{{"gold", "int", 25}, {"loot", "string", "Brass Lantern"}}},
		{ID: 3, Name: "signpost", Type: "trigger", GID: markersFirst + markerSign,
			X: px(signTile[0]), Y: px(signTile[1] + 1), Width: tilePx, Height: tilePx, Visible: true,
			Properties: []prop{{"message", "string", "East: the old pond. Beyond it, the ruins."}}},
		{ID: 4, Name: "bridge_zone", Type: "trigger",
			X: px(int(pondC[0] - pondR[0] - 1)), Y: px(bridgeY), Width: px(int(2*pondR[0]) + 2), Height: tilePx, Visible: true,
			Properties: []prop{{"message", "string", "The bridge creaks underfoot."}}},
		{ID: 5, Name: "ruins_gate", Type: "trigger",
			X: px(chestTile[0]), Y: px(ruins[1]), Width: tilePx, Height: tilePx, Visible: true,
			Properties: []prop{{"message", "string", "Ancient ruins. Something glints inside."}}},
	}
	collision := []object{
		// The signpost is a tile object, and tile objects carry no collision,
		// so its post gets an explicit shape here.
		{ID: 6, Name: "signpost_post", X: px(signTile[0]) + 6, Y: px(signTile[1]) + 8, Width: 4, Height: 8, Visible: true},
	}
	return tiledMap{
		Type: "map", Version: "1.10", TiledVersion: "1.10.2",
		Orientation: "orthogonal", RenderOrder: "right-down",
		Width: mapW, Height: mapH, TileWidth: tilePx, TileHeight: tilePx,
		BackgroundColor: "#1d2b1f", NextLayerID: 6, NextObjectID: 7,
		Properties: []prop{
			{"ambient", "color", "#ffe8f0d8"},
			{"daylight", "float", 0.9},
			{"seed", "int", seed},
			{"title", "string", "Pondside Ruins"},
		},
		Tilesets: []any{tilesetRef{FirstGID: terrainFirst, Source: TerrainFile}, markers},
		Layers: []layer{
			{ID: 1, Name: "terrain", Type: "group", Visible: true, Opacity: 1, Layers: []layer{
				{ID: 2, Name: "ground", Type: "tilelayer", Visible: true, Opacity: 1,
					Width: mapW, Height: mapH, Data: ground},
				{ID: 3, Name: "decor", Type: "tilelayer", Visible: true, Opacity: 1,
					Width: mapW, Height: mapH, Encoding: "base64", Compression: "zlib", Data: decorBase64},
			}},
			{ID: 4, Name: "entities", Type: "objectgroup", Visible: true, Opacity: 1, DrawOrder: "topdown", Objects: entities},
			{ID: 5, Name: "collision", Type: "objectgroup", Class: "collision", Visible: false, Opacity: 1, DrawOrder: "index", Objects: collision},
		},
	}
}
