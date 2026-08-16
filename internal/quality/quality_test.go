package quality

import (
	"math"
	"strings"
	"testing"

	"Tessera/internal/delaunay"
	"Tessera/internal/geom"
)

// close fails the test unless two numbers agree to the given tolerance.
func close(t *testing.T, got, want, tolerance float64, label string) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Fatalf("%s = %g, want %g within %g", label, got, want, tolerance)
	}
}

// lattice returns a side by side square lattice.
func lattice(side int) []geom.Point {
	out := make([]geom.Point, 0, side*side)
	for row := 0; row < side; row++ {
		for column := 0; column < side; column++ {
			out = append(out, geom.Point{X: float64(column), Y: float64(row)})
		}
	}
	return out
}

// meshOf triangulates a point set or fails the test.
func meshOf(t *testing.T, points []geom.Point) *delaunay.Mesh {
	t.Helper()
	mesh, err := delaunay.Triangulate(points)
	if err != nil {
		t.Fatalf("Triangulate: %v", err)
	}
	return mesh
}

func TestOfPointsOnAnEquilateralTriangle(t *testing.T) {
	metrics, err := OfPoints(
		geom.Point{X: 0, Y: 0},
		geom.Point{X: 1, Y: 0},
		geom.Point{X: 0.5, Y: math.Sqrt(3) / 2})
	if err != nil {
		t.Fatalf("OfPoints: %v", err)
	}
	close(t, metrics.MinAngle, 60, 1e-9, "smallest angle")
	close(t, metrics.MaxAngle, 60, 1e-9, "largest angle")
	close(t, metrics.AspectRatio, 1, 1e-12, "aspect ratio")
	close(t, metrics.Area, math.Sqrt(3)/4, 1e-15, "area")
	// The circumradius of a unit equilateral triangle is one over root three.
	close(t, metrics.RadiusEdgeRatio, 1/math.Sqrt(3), 1e-12, "radius-edge ratio")
	if metrics.AreaUnderflowed {
		t.Fatalf("an equilateral triangle has an area that fits")
	}
}

func TestOfPointsOnARightTriangle(t *testing.T) {
	metrics, err := OfPoints(
		geom.Point{X: 0, Y: 0},
		geom.Point{X: 1, Y: 0},
		geom.Point{X: 0, Y: 1})
	if err != nil {
		t.Fatalf("OfPoints: %v", err)
	}
	close(t, metrics.MinAngle, 45, 1e-9, "smallest angle")
	close(t, metrics.MaxAngle, 90, 1e-9, "largest angle")
	close(t, metrics.Area, 0.5, 1e-15, "area")
	close(t, metrics.ShortestEdge, 1, 1e-15, "shortest edge")
	close(t, metrics.LongestEdge, math.Sqrt2, 1e-15, "longest edge")
	close(t, metrics.AspectRatio, math.Sqrt2, 1e-12, "aspect ratio")
	close(t, metrics.RadiusEdgeRatio, math.Sqrt2/2, 1e-12, "radius-edge ratio")
}

func TestOfPointsOnASliver(t *testing.T) {
	metrics, err := OfPoints(
		geom.Point{X: 0, Y: 0},
		geom.Point{X: 100, Y: 0},
		geom.Point{X: 50, Y: 0.5})
	if err != nil {
		t.Fatalf("OfPoints: %v", err)
	}
	if metrics.MinAngle > 1 {
		t.Fatalf("the smallest angle of a sliver = %g degrees, want less than one", metrics.MinAngle)
	}
	if metrics.MaxAngle < 170 {
		t.Fatalf("the largest angle of a sliver = %g degrees, want more than 170", metrics.MaxAngle)
	}
	if metrics.AspectRatio < 1.9 {
		t.Fatalf("the aspect ratio of a sliver = %g, want more than 1.9", metrics.AspectRatio)
	}
	if metrics.RadiusEdgeRatio < 1 {
		t.Fatalf("the radius-edge ratio of a sliver = %g, want more than 1", metrics.RadiusEdgeRatio)
	}
}

func TestOfPointsRejectsShapelessTriangles(t *testing.T) {
	cases := []struct {
		label   string
		a, b, c geom.Point
	}{
		{"collinear", geom.Point{X: 0}, geom.Point{X: 1}, geom.Point{X: 2}},
		{"repeated point", geom.Point{X: 0}, geom.Point{X: 0}, geom.Point{X: 1, Y: 1}},
	}
	for _, item := range cases {
		if _, err := OfPoints(item.a, item.b, item.c); err == nil {
			t.Fatalf("%s: OfPoints = nil error, want a failure", item.label)
		}
	}
}

func TestOfATriangleOfAMesh(t *testing.T) {
	mesh := meshOf(t, lattice(3))
	metrics, err := Of(mesh, 0)
	if err != nil {
		t.Fatalf("Of: %v", err)
	}
	close(t, metrics.Area, 0.5, 1e-15, "the area of a lattice triangle")
	close(t, metrics.MinAngle, 45, 1e-9, "the smallest angle of a lattice triangle")
	for _, index := range []int{-1, len(mesh.Triangles)} {
		if _, err := Of(mesh, index); err == nil {
			t.Fatalf("Of(%d) = nil error, want a failure", index)
		}
	}
}

func TestSummariseALattice(t *testing.T) {
	mesh := meshOf(t, lattice(5))
	summary, err := Summarise(mesh)
	if err != nil {
		t.Fatalf("Summarise: %v", err)
	}
	if summary.Triangles != 32 {
		t.Fatalf("Triangles = %d, want 32", summary.Triangles)
	}
	// Every triangle of a lattice is half a unit square, so every angle is 45 or 90.
	close(t, summary.MinAngle, 45, 1e-9, "smallest angle")
	close(t, summary.MaxAngle, 90, 1e-9, "largest angle")
	close(t, summary.MeanMinAngle, 45, 1e-9, "mean smallest angle")
	close(t, summary.MinArea, 0.5, 1e-15, "smallest area")
	close(t, summary.MaxArea, 0.5, 1e-15, "largest area")
	close(t, summary.TotalArea, 16, 1e-12, "total area")
	close(t, summary.WorstAspect, math.Sqrt2, 1e-12, "worst aspect ratio")
	close(t, summary.WorstRadiusEdge, math.Sqrt2/2, 1e-12, "worst radius-edge ratio")
	if summary.Worst < 0 || summary.Worst >= summary.Triangles {
		t.Fatalf("Worst = %d, which is not a triangle", summary.Worst)
	}
	if summary.Underflowed != 0 {
		t.Fatalf("Underflowed = %d, want 0 for a lattice", summary.Underflowed)
	}
}

func TestSummariseRejectsAnEmptyMesh(t *testing.T) {
	if _, err := Summarise(nil); err == nil {
		t.Fatalf("a missing mesh must be reported")
	}
	if _, err := Summarise(&delaunay.Mesh{Points: lattice(3)}); err == nil {
		t.Fatalf("a mesh with no triangles must be reported")
	}
}

func TestHistogram(t *testing.T) {
	mesh := meshOf(t, lattice(5))
	counts, err := Histogram(mesh, 6)
	if err != nil {
		t.Fatalf("Histogram: %v", err)
	}
	if len(counts) != 6 {
		t.Fatalf("the histogram has %d bucket(s), want 6", len(counts))
	}
	total := 0
	for _, count := range counts {
		total += count
	}
	if total != len(mesh.Triangles) {
		t.Fatalf("the buckets hold %d triangle(s) of %d", total, len(mesh.Triangles))
	}
	// Every smallest angle is 45 degrees, which lands in the bucket from 40 to 50.
	if counts[4] != len(mesh.Triangles) {
		t.Fatalf("the 40 to 50 degree bucket holds %d of %d", counts[4], len(mesh.Triangles))
	}
	single, err := Histogram(mesh, 1)
	if err != nil {
		t.Fatalf("Histogram: %v", err)
	}
	if single[0] != len(mesh.Triangles) {
		t.Fatalf("one bucket holds %d of %d", single[0], len(mesh.Triangles))
	}
	if _, err := Histogram(mesh, 0); err == nil {
		t.Fatalf("a histogram of no buckets must be reported")
	}
	if _, err := Histogram(nil, 4); err == nil {
		t.Fatalf("a histogram of no mesh must be reported")
	}
}

func TestSlivers(t *testing.T) {
	mesh := meshOf(t, lattice(4))
	slivers, err := Slivers(mesh, 15)
	if err != nil {
		t.Fatalf("Slivers: %v", err)
	}
	if len(slivers) != 0 {
		t.Fatalf("a lattice has no slivers, got %v", slivers)
	}
	all, err := Slivers(mesh, 50)
	if err != nil {
		t.Fatalf("Slivers: %v", err)
	}
	if len(all) != len(mesh.Triangles) {
		t.Fatalf("every lattice triangle is below 50 degrees, got %d of %d",
			len(all), len(mesh.Triangles))
	}
	thin := meshOf(t, []geom.Point{
		{X: 0, Y: 0}, {X: 100, Y: 0}, {X: 50, Y: 0.5}, {X: 25, Y: 0.2},
	})
	found, err := Slivers(thin, 15)
	if err != nil {
		t.Fatalf("Slivers: %v", err)
	}
	if len(found) == 0 {
		t.Fatalf("a mesh of nearly flat triangles has slivers")
	}
	for _, threshold := range []float64{0, -1, 60, 90} {
		if _, err := Slivers(mesh, threshold); err == nil {
			t.Fatalf("Slivers(%g) = nil error, want a failure", threshold)
		}
	}
	if _, err := Slivers(nil, 15); err == nil {
		t.Fatalf("a missing mesh must be reported")
	}
}

func TestDescribe(t *testing.T) {
	metrics, err := OfPoints(geom.Point{X: 0, Y: 0}, geom.Point{X: 1, Y: 0}, geom.Point{X: 0, Y: 1})
	if err != nil {
		t.Fatalf("OfPoints: %v", err)
	}
	got := metrics.Describe()
	for _, fragment := range []string{"area 0.5", "angles 45 to 90 degrees", "aspect"} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("Describe = %q, which is missing %q", got, fragment)
		}
	}
	mesh := meshOf(t, lattice(3))
	summary, err := Summarise(mesh)
	if err != nil {
		t.Fatalf("Summarise: %v", err)
	}
	rendered := summary.Describe()
	for _, fragment := range []string{"8 triangle(s)", "smallest angle 45", "largest angle 90"} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("Describe = %q, which is missing %q", rendered, fragment)
		}
	}
}
