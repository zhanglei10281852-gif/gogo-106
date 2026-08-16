// Package hull computes the convex hull of a point set by the monotone chain.
//
// The algorithm is the one worth knowing because it has no special cases beyond the
// one that matters: sort the points, sweep once building the lower chain and once
// building the upper chain, and pop a vertex whenever the last three do not turn left.
// Whether they turn left is decided by the exact predicate, so a point that lies a
// nanometre off the line is treated the same way every time it is examined.
package hull

import (
	"fmt"

	"Tessera/internal/exact"
	"Tessera/internal/geom"
)

// Hull returns the vertices of the convex hull counterclockwise, starting at the
// lexicographically smallest point. Repeated points are ignored, and a set whose
// points are all collinear yields the two ends of the segment they span.
func Hull(points []geom.Point) ([]geom.Point, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("the hull of no points is undefined")
	}
	for index, point := range points {
		if !point.IsFinite() {
			return nil, fmt.Errorf("point %d is not finite", index+1)
		}
	}
	sorted := geom.Deduplicate(geom.SortedLexicographic(points))
	if len(sorted) == 1 {
		return sorted, nil
	}
	if len(sorted) == 2 {
		return sorted, nil
	}
	if geom.AllCollinear(sorted) {
		return []geom.Point{sorted[0], sorted[len(sorted)-1]}, nil
	}
	lower := chain(sorted)
	reversed := make([]geom.Point, len(sorted))
	for index, point := range sorted {
		reversed[len(sorted)-1-index] = point
	}
	upper := chain(reversed)
	out := make([]geom.Point, 0, len(lower)+len(upper))
	out = append(out, lower[:len(lower)-1]...)
	out = append(out, upper[:len(upper)-1]...)
	return out, nil
}

// chain builds one monotone chain over points that are already sorted.
func chain(points []geom.Point) []geom.Point {
	out := make([]geom.Point, 0, len(points))
	for _, point := range points {
		for len(out) >= 2 {
			if geom.Orientation(out[len(out)-2], out[len(out)-1], point) == exact.CounterClockwise {
				break
			}
			out = out[:len(out)-1]
		}
		out = append(out, point)
	}
	return out
}

// Perimeter returns the length of the hull boundary.
func Perimeter(vertices []geom.Point) float64 {
	if len(vertices) < 3 {
		if len(vertices) == 2 {
			return 2 * vertices[0].Dist(vertices[1])
		}
		return 0
	}
	return geom.PolygonPerimeter(vertices)
}

// Area returns the area enclosed by the hull, which is zero for a degenerate hull.
func Area(vertices []geom.Point) float64 {
	if len(vertices) < 3 {
		return 0
	}
	return geom.PolygonArea(vertices)
}

// Degenerate reports whether the hull encloses no area.
func Degenerate(vertices []geom.Point) bool { return len(vertices) < 3 }

// Contains reports whether a point lies in the closed hull.
func Contains(vertices []geom.Point, point geom.Point) bool {
	if len(vertices) < 3 {
		return false
	}
	for index := range vertices {
		a := vertices[index]
		b := vertices[(index+1)%len(vertices)]
		if geom.Orientation(a, b, point) == exact.Clockwise {
			return false
		}
	}
	return true
}

// IndexIn returns, for every hull vertex, its position in the original point list,
// which is what lets a report name the input point a hull vertex came from.
func IndexIn(points []geom.Point, vertices []geom.Point) ([]int, error) {
	out := make([]int, 0, len(vertices))
	for _, vertex := range vertices {
		found := -1
		for index, point := range points {
			if point.Equal(vertex) {
				found = index
				break
			}
		}
		if found < 0 {
			return nil, fmt.Errorf("hull vertex %s is not one of the input points", vertex)
		}
		out = append(out, found)
	}
	return out, nil
}
