package tiled

import (
	"encoding/json"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
)

// The raw* types mirror Tiled's JSON format field for field. They are kept
// private: decoding into them is mechanical, and the loader then converts
// them into the validated public model. That split keeps format quirks
// (legacy field names, optional values with non-zero defaults) out of the
// types game code sees.

type rawMap struct {
	Type            string        `json:"type"`
	Orientation     string        `json:"orientation"`
	RenderOrder     string        `json:"renderorder"`
	Width           int           `json:"width"`
	Height          int           `json:"height"`
	TileWidth       int           `json:"tilewidth"`
	TileHeight      int           `json:"tileheight"`
	Infinite        bool          `json:"infinite"`
	BackgroundColor string        `json:"backgroundcolor"`
	Class           string        `json:"class"`
	Properties      []rawProperty `json:"properties"`
	Tilesets        []rawTileset  `json:"tilesets"`
	Layers          []rawLayer    `json:"layers"`
}

type rawTileset struct {
	FirstGID    uint32        `json:"firstgid"`
	Source      string        `json:"source"`
	Name        string        `json:"name"`
	Class       string        `json:"class"`
	TileWidth   int           `json:"tilewidth"`
	TileHeight  int           `json:"tileheight"`
	Margin      int           `json:"margin"`
	Spacing     int           `json:"spacing"`
	TileCount   int           `json:"tilecount"`
	Columns     int           `json:"columns"`
	Image       string        `json:"image"`
	ImageWidth  int           `json:"imagewidth"`
	ImageHeight int           `json:"imageheight"`
	TileOffset  *geom.Vec     `json:"tileoffset"`
	Properties  []rawProperty `json:"properties"`
	Tiles       []rawTile     `json:"tiles"`
}

type rawTile struct {
	ID          int           `json:"id"`
	Type        string        `json:"type"` // Tiled < 1.9 name for class
	Class       string        `json:"class"`
	Probability *float64      `json:"probability"`
	Image       string        `json:"image"`
	Properties  []rawProperty `json:"properties"`
	ObjectGroup *rawLayer     `json:"objectgroup"`
	Animation   []rawFrame    `json:"animation"`
}

type rawFrame struct {
	TileID   int `json:"tileid"`
	Duration int `json:"duration"` // milliseconds
}

type rawLayer struct {
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Class       string          `json:"class"`
	Visible     *bool           `json:"visible"`
	Opacity     *float64        `json:"opacity"`
	OffsetX     float64         `json:"offsetx"`
	OffsetY     float64         `json:"offsety"`
	ParallaxX   *float64        `json:"parallaxx"`
	ParallaxY   *float64        `json:"parallaxy"`
	TintColor   string          `json:"tintcolor"`
	Properties  []rawProperty   `json:"properties"`
	Width       int             `json:"width"`
	Height      int             `json:"height"`
	Encoding    string          `json:"encoding"`
	Compression string          `json:"compression"`
	Data        json.RawMessage `json:"data"`
	Chunks      []rawChunk      `json:"chunks"`
	DrawOrder   string          `json:"draworder"`
	Color       string          `json:"color"`
	Objects     []rawObject     `json:"objects"`
	Image       string          `json:"image"`
	RepeatX     bool            `json:"repeatx"`
	RepeatY     bool            `json:"repeaty"`
	Layers      []rawLayer      `json:"layers"`
}

type rawChunk struct {
	X      int             `json:"x"`
	Y      int             `json:"y"`
	Width  int             `json:"width"`
	Height int             `json:"height"`
	Data   json.RawMessage `json:"data"`
}

type rawObject struct {
	ID         int           `json:"id"`
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Class      string        `json:"class"`
	X          float64       `json:"x"`
	Y          float64       `json:"y"`
	Width      float64       `json:"width"`
	Height     float64       `json:"height"`
	Rotation   float64       `json:"rotation"`
	Visible    *bool         `json:"visible"`
	GID        uint32        `json:"gid"`
	Point      bool          `json:"point"`
	Ellipse    bool          `json:"ellipse"`
	Polygon    []geom.Vec    `json:"polygon"`
	Polyline   []geom.Vec    `json:"polyline"`
	Text       *rawText      `json:"text"`
	Template   string        `json:"template"`
	Properties []rawProperty `json:"properties"`
}

type rawText struct {
	Text string `json:"text"`
}

func boolOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func floatOr(p *float64, def float64) float64 {
	if p == nil {
		return def
	}
	return *p
}

// firstNonEmpty prefers the modern field name and falls back to the legacy one.
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
