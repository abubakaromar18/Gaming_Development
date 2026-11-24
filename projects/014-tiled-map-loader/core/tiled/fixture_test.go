package tiled_test

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"testing"
	"testing/fstest"
)

// baseMap is a small map exercising most of the format: an external tileset
// in a sub-directory, an embedded tileset, a group with CSV and base64+zlib
// layers, every object shape, and every property type.
const baseMap = `{
 "type": "map", "orientation": "orthogonal", "renderorder": "right-down",
 "width": 4, "height": 3, "tilewidth": 16, "tileheight": 16, "infinite": false,
 "backgroundcolor": "#336699",
 "properties": [
  {"name": "title", "type": "string", "value": "Fixture"},
  {"name": "gravity", "type": "float", "value": 9.5},
  {"name": "lives", "type": "int", "value": 3},
  {"name": "hard", "type": "bool", "value": true},
  {"name": "tint", "type": "color", "value": "#80ff0000"},
  {"name": "music", "type": "file", "value": "theme.ogg"},
  {"name": "boss", "type": "object", "value": 7},
  {"name": "stats", "type": "class", "propertytype": "Stats", "value": {"hp": 10}}
 ],
 "tilesets": [
  {"firstgid": 10, "name": "items", "tilewidth": 16, "tileheight": 16, "tilecount": 2, "columns": 2,
   "image": "items.png", "imagewidth": 32, "imageheight": 16},
  {"firstgid": 1, "source": "sets/terrain.tsj"}
 ],
 "layers": [
  {"id": 1, "name": "world", "type": "group", "opacity": 0.5, "offsetx": 4, "offsety": 2, "parallaxx": 0.5, "visible": true,
   "layers": [
    {"id": 2, "name": "ground", "type": "tilelayer", "width": 4, "height": 3, "opacity": 0.5, "offsetx": 1, "visible": true,
     "data": [1, 1, 1, 1, 1, 2, 2147483650, 1, 3, 3, 4, 3]},
    {"id": 3, "name": "decor", "type": "tilelayer", "width": 4, "height": 3, "visible": false,
     "encoding": "base64", "compression": "zlib", "data": "DECOR"}
   ]},
  {"id": 4, "name": "things", "type": "objectgroup", "objects": [
   {"id": 1, "name": "start", "type": "spawn", "x": 8, "y": 8, "point": true},
   {"id": 2, "name": "coin", "class": "pickup", "gid": 10, "x": 16, "y": 32, "width": 16, "height": 16,
    "properties": [{"name": "value", "type": "int", "value": 5}]},
   {"id": 3, "name": "zone", "type": "trigger", "x": 0, "y": 0, "width": 32, "height": 16, "rotation": 90},
   {"id": 4, "name": "hill", "x": 10, "y": 10, "polygon": [{"x": 0, "y": 0}, {"x": 20, "y": -5}, {"x": 5, "y": 15}]},
   {"id": 5, "name": "rope", "x": 0, "y": 0, "polyline": [{"x": 0, "y": 0}, {"x": 3, "y": 4}]},
   {"id": 6, "name": "pond", "x": 2, "y": 2, "width": 4, "height": 6, "ellipse": true},
   {"id": 7, "name": "label", "x": 1, "y": 1, "width": 30, "height": 10, "text": {"text": "Hello"}, "visible": false}
  ]}
 ]
}`

const terrainTileset = `{
 "name": "terrain", "tilewidth": 16, "tileheight": 16, "tilecount": 4, "columns": 2,
 "margin": 1, "spacing": 2, "image": "../img/terrain.png", "imagewidth": 36, "imageheight": 36,
 "tiles": [
  {"id": 1, "type": "wall", "properties": [{"name": "solid", "type": "bool", "value": true}]},
  {"id": 2, "class": "water", "animation": [{"tileid": 2, "duration": 100}, {"tileid": 3, "duration": 200}]},
  {"id": 3, "probability": 0.25, "objectgroup": {"type": "objectgroup", "objects": [{"id": 1, "x": 2, "y": 4, "width": 8, "height": 12}]}}
 ]
}`

// decorGIDs is the content of the base64 layer: a flipped tile and an item.
var decorGIDs = []uint32{0, 0, 0, 0, 0, 0x40000000 | 3, 0, 11, 0, 0, 0, 0}

func encodeZlib(t *testing.T, gids []uint32) string {
	t.Helper()
	raw := make([]byte, 4*len(gids))
	for i, g := range gids {
		binary.LittleEndian.PutUint32(raw[i*4:], g)
	}
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// fixture returns the base map as a generic JSON tree so tests can mutate
// one field to provoke a specific error.
func fixture(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(baseMap), &doc); err != nil {
		t.Fatal(err)
	}
	group := layers(doc)[0]
	children := group["layers"].([]any)
	children[1].(map[string]any)["data"] = encodeZlib(t, decorGIDs)
	return doc
}

func layers(doc map[string]any) []map[string]any {
	var out []map[string]any
	for _, l := range doc["layers"].([]any) {
		out = append(out, l.(map[string]any))
	}
	return out
}

func groundLayer(doc map[string]any) map[string]any {
	return layers(doc)[0]["layers"].([]any)[0].(map[string]any)
}

func fsWith(t *testing.T, doc map[string]any) fstest.MapFS {
	t.Helper()
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return fstest.MapFS{
		"maps/level.tmj":        {Data: data},
		"maps/sets/terrain.tsj": {Data: []byte(terrainTileset)},
	}
}
