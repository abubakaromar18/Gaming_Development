package tiled

import "github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"

// Shape is the geometric kind of an Object.
type Shape int

// Object shapes Tiled can export.
const (
	ShapeRectangle Shape = iota
	ShapeEllipse
	ShapePoint
	ShapePolygon
	ShapePolyline
	ShapeTile // a tile placed as a free object (GID is set)
	ShapeText
)

var shapeNames = [...]string{"rectangle", "ellipse", "point", "polygon", "polyline", "tile", "text"}

func (s Shape) String() string {
	if int(s) < len(shapeNames) {
		return shapeNames[s]
	}
	return "unknown"
}

// Object is anything placed in an object layer: spawn points, trigger zones,
// collision polygons, free-placed tiles. Designers use them to attach
// gameplay meaning to a map without touching code.
type Object struct {
	ID       int
	Name     string
	Class    string
	X, Y     float64 // anchor; top-left for most shapes, bottom-left for tiles
	Width    float64
	Height   float64
	Rotation float64 // degrees clockwise around (X, Y)
	Visible  bool
	Shape    Shape
	GID      GID        // tile objects only
	Points   []geom.Vec // polygon/polyline vertices relative to (X, Y)
	Text     string     // text objects only
	// Properties are the object's own custom properties.
	Properties Properties
}

// Bounds returns the axis-aligned bounding box of the object in map pixels,
// accounting for rotation and Tiled's bottom-left anchor on tile objects.
func (o *Object) Bounds() geom.Rect {
	var local []geom.Vec
	switch o.Shape {
	case ShapePoint:
		return geom.Rect{X: o.X, Y: o.Y}
	case ShapePolygon, ShapePolyline:
		local = o.Points
	case ShapeTile:
		local = geom.Rect{Y: -o.Height, W: o.Width, H: o.Height}.Corners()
	default:
		local = geom.Rect{W: o.Width, H: o.Height}.Corners()
	}
	rot := geom.Rotation(o.Rotation)
	pts := make([]geom.Vec, len(local))
	for i, p := range local {
		pts[i] = rot.Apply(p).Add(geom.Vec{X: o.X, Y: o.Y})
	}
	return geom.RectFromPoints(pts)
}
