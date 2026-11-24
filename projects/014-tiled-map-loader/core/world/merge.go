package world

import "image"

// MergeCells covers the true cells of a row-major w×h grid with
// non-overlapping rectangles, in cell units.
//
// It is the greedy "meshing" pass used by tile engines and voxel games: grow
// a run to the right, then grow that run downward while every cell below is
// also free. A wall of 40 solid tiles becomes one box instead of 40, which
// shrinks collision lists and removes the internal seams that can snag a body
// sliding along a row of separate boxes. Greedy is not guaranteed to find the
// minimum number of rectangles, but it is linear-time and close in practice.
func MergeCells(solid []bool, w, h int) []image.Rectangle {
	used := make([]bool, len(solid))
	free := func(x, y int) bool { i := y*w + x; return solid[i] && !used[i] }
	var rects []image.Rectangle
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !free(x, y) {
				continue
			}
			x1 := x + 1
			for x1 < w && free(x1, y) {
				x1++
			}
			y1 := y + 1
		grow:
			for y1 < h {
				for xx := x; xx < x1; xx++ {
					if !free(xx, y1) {
						break grow
					}
				}
				y1++
			}
			for yy := y; yy < y1; yy++ {
				for xx := x; xx < x1; xx++ {
					used[yy*w+xx] = true
				}
			}
			rects = append(rects, image.Rect(x, y, x1, y1))
		}
	}
	return rects
}
