package hull

import (
	"math"
	"testing"

	"Tessera/internal/geom"
)

// square returns the corners of a two by two square with a point in the middle.
func square() []geom.Point {
	return []geom.Point{
		{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}, {X: 1, Y: 1},
	}
}

func TestHullOfASquareWithAnInteriorPoint(t *testing.T) {
	vertices, err := Hull(square())
	if err != nil {
		t.Fatalf("Hull: %v", err)
	}
	if len(vertices) != 4 {
		t.Fatalf("hull vertices = %d, want 4: %v", len(vertices), vertices)
	}
	if vertices[0] != (geom.Point{X: 0, Y: 0}) {
		t.Fatalf("the walk starts at the smallest point, got %s", vertices[0])
	}
	if got := Area(vertices); got != 4 {
		t.Fatalf("Area = %g, want 4", got)
	}
	if got := Perimeter(vertices); got != 8 {
		t.Fatalf("Perimeter = %g, want 8", got)
	}
	if !geom.IsConvex(vertices) {
		t.Fatalf("the hull must be convex: %v", vertices)
	}
	if Degenerate(vertices) {
		t.Fatalf("a square hull is not degenerate")
	}
}

func TestHullDropsPointsOnAnEdge(t *testing.T) {
	points := []geom.Point{
		{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0},
		{X: 2, Y: 2}, {X: 1, Y: 2}, {X: 0, Y: 2},
	}
	vertices, err := Hull(points)
	if err != nil {
		t.Fatalf("Hull: %v", err)
	}
	if len(vertices) != 4 {
		t.Fatalf("a point in the middle of an edge is not a corner, got %v", vertices)
	}
	if got := Area(vertices); got != 4 {
		t.Fatalf("Area = %g, want 4", got)
	}
}

func TestHullIgnoresRepeatedPoints(t *testing.T) {
	points := []geom.Point{
		{X: 0, Y: 0}, {X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 0},
		{X: 2, Y: 2}, {X: 0, Y: 2}, {X: 0, Y: 2},
	}
	vertices, err := Hull(points)
	if err != nil {
		t.Fatalf("Hull: %v", err)
	}
	if len(vertices) != 4 {
		t.Fatalf("hull vertices = %d, want 4: %v", len(vertices), vertices)
	}
}

func TestHullOfCollinearPoints(t *testing.T) {
	points := []geom.Point{{X: 3, Y: 6}, {X: 0, Y: 0}, {X: 1, Y: 2}, {X: 2, Y: 4}}
	vertices, err := Hull(points)
	if err != nil {
		t.Fatalf("Hull: %v", err)
	}
	if len(vertices) != 2 {
		t.Fatalf("the hull of a segment has two ends, got %v", vertices)
	}
	if !Degenerate(vertices) {
		t.Fatalf("a segment hull is degenerate")
	}
	if got := Area(vertices); got != 0 {
		t.Fatalf("Area = %g, want 0", got)
	}
	if got := Perimeter(vertices); math.Abs(got-2*math.Sqrt(45)) > 1e-12 {
		t.Fatalf("Perimeter = %g, want twice the length of the segment", got)
	}
}

func TestHullOfOneAndTwoPoints(t *testing.T) {
	one, err := Hull([]geom.Point{{X: 1, Y: 1}})
	if err != nil {
		t.Fatalf("Hull: %v", err)
	}
	if len(one) != 1 {
		t.Fatalf("the hull of one point is that point, got %v", one)
	}
	if got := Perimeter(one); got != 0 {
		t.Fatalf("Perimeter = %g, want 0", got)
	}
	two, err := Hull([]geom.Point{{X: 1, Y: 1}, {X: 0, Y: 0}})
	if err != nil {
		t.Fatalf("Hull: %v", err)
	}
	if len(two) != 2 {
		t.Fatalf("the hull of two points is both, got %v", two)
	}
	if two[0] != (geom.Point{X: 0, Y: 0}) {
		t.Fatalf("the hull starts at the smallest point, got %s", two[0])
	}
}

func TestHullRejectsBadInput(t *testing.T) {
	if _, err := Hull(nil); err == nil {
		t.Fatalf("the hull of no points must be reported")
	}
	if _, err := Hull([]geom.Point{{X: 0}, {X: math.NaN()}}); err == nil {
		t.Fatalf("a point that is not finite must be reported")
	}
}

func TestContains(t *testing.T) {
	vertices, err := Hull(square())
	if err != nil {
		t.Fatalf("Hull: %v", err)
	}
	for _, point := range []geom.Point{{X: 1, Y: 1}, {X: 0, Y: 0}, {X: 2, Y: 1}, {X: 0.01, Y: 1.99}} {
		if !Contains(vertices, point) {
			t.Fatalf("the hull must contain %s", point)
		}
	}
	for _, point := range []geom.Point{{X: -0.5, Y: 1}, {X: 3, Y: 3}, {X: 1, Y: 2.0001}} {
		if Contains(vertices, point) {
			t.Fatalf("the hull must not contain %s", point)
		}
	}
	if Contains([]geom.Point{{X: 0}, {X: 1}}, geom.Point{X: 0.5}) {
		t.Fatalf("a degenerate hull contains nothing")
	}
}

func TestIndexIn(t *testing.T) {
	points := square()
	vertices, err := Hull(points)
	if err != nil {
		t.Fatalf("Hull: %v", err)
	}
	indices, err := IndexIn(points, vertices)
	if err != nil {
		t.Fatalf("IndexIn: %v", err)
	}
	if len(indices) != len(vertices) {
		t.Fatalf("IndexIn returned %d index(es) for %d vertex(es)", len(indices), len(vertices))
	}
	for position, index := range indices {
		if index < 0 || index >= len(points) {
			t.Fatalf("index %d is not a position in the point list", index)
		}
		if !points[index].Equal(vertices[position]) {
			t.Fatalf("index %d names %s, but hull vertex %d is %s",
				index, points[index], position, vertices[position])
		}
	}
	if _, err := IndexIn(points, []geom.Point{{X: 9, Y: 9}}); err == nil {
		t.Fatalf("a vertex that is not an input point must be reported")
	}
}
