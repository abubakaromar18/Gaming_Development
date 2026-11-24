package world_test

import (
	"encoding/json"
	"testing"
	"testing/fstest"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/tiled"
)

const tile = 16

// buildMap makes a map from ASCII rows:
//
//	'.' empty   '#' solid wall   'L' tile whose collision shape is its left
//	half        'R' the same tile flipped horizontally   'w' animated water
//
// extraLayers are appended as-is after the tile layer.
func buildMap(t *testing.T, rows []string, extraLayers ...map[string]any) *tiled.Map {
	t.Helper()
	legend := map[rune]uint32{'.': 0, '#': 1, 'L': 2, 'R': 2 | 0x80000000, 'w': 3}
	var data []uint32
	for _, row := range rows {
		for _, r := range row {
			gid, ok := legend[r]
			if !ok {
				t.Fatalf("unknown map glyph %q", r)
			}
			data = append(data, gid)
		}
	}
	layers := []any{map[string]any{"name": "ground", "type": "tilelayer", "width": len(rows[0]), "height": len(rows), "data": data}}
	for _, l := range extraLayers {
		layers = append(layers, l)
	}
	doc := map[string]any{
		"type": "map", "orientation": "orthogonal",
		"width": len(rows[0]), "height": len(rows), "tilewidth": tile, "tileheight": tile,
		"tilesets": []any{map[string]any{
			"firstgid": 1, "name": "test", "tilewidth": tile, "tileheight": tile, "tilecount": 4, "columns": 4,
			"image": "test.png", "imagewidth": 4 * tile, "imageheight": tile,
			"tiles": []any{
				map[string]any{"id": 0, "properties": []any{map[string]any{"name": "solid", "type": "bool", "value": true}}},
				map[string]any{"id": 1, "objectgroup": map[string]any{"type": "objectgroup", "objects": []any{
					map[string]any{"id": 1, "x": 0, "y": 0, "width": 8, "height": tile},
				}}},
				map[string]any{"id": 2, "animation": []any{
					map[string]any{"tileid": 2, "duration": 100},
					map[string]any{"tileid": 3, "duration": 200},
				}},
			},
		}},
		"layers": layers,
	}
	data2, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	m, err := tiled.Load(fstest.MapFS{"m.tmj": {Data: data2}}, "m.tmj")
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func objectLayer(name, class string, objects ...map[string]any) map[string]any {
	objs := make([]any, len(objects))
	for i, o := range objects {
		o["id"] = i + 1
		objs[i] = o
	}
	return map[string]any{"name": name, "class": class, "type": "objectgroup", "objects": objs}
}
