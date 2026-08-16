package delaunay

import (
	"math"
	"strings"
	"testing"

	"Tessera/internal/exact"
	"Tessera/internal/geom"
)

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

// circle returns count points spaced evenly on the unit circle.
func circle(count int) []geom.Point {
	out := make([]geom.Point, 0, count)
	for index := 0; index < count; index++ {
		angle := 2 * math.Pi * float64(index) / float64(count)
		out = append(out, geom.Point{X: math.Cos(angle), Y: math.Sin(angle)})
	}
	return out
}

func TestTriangleHelpers(t *testing.T) {
	triangle := Triangle{A: 4, B: 1, C: 7}
	if got := triangle.Vertices(); got != [3]int{4, 1, 7} {
		t.Fatalf("Vertices = %v", got)
	}
	if !triangle.Has(1) || triangle.Has(2) {
		t.Fatalf("Has disagrees with the vertices")
	}
	if got := triangle.Edges(); got != [3][2]int{{4, 1}, {1, 7}, {7, 4}} {
		t.Fatalf("Edges = %v", got)
	}
	if got := triangle.Key(); got != [3]int{1, 4, 7} {
		t.Fatalf("Key = %v, want the vertices in order", got)
	}
	if got, want := triangle.String(), "(4 1 7)"; got != want {
		t.Fatalf("String = %q, want %q", got, want)
	}
}

func TestTriangulateASquareWithACentre(t *testing.T) {
	points := []geom.Point{
		{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}, {X: 1, Y: 1},
	}
	mesh, err := Triangulate(points)
	if err != nil {
		t.Fatalf("Triangulate: %v", err)
	}
	if len(mesh.Triangles) != 4 {
		t.Fatalf("triangles = %d, want 4: %v", len(mesh.Triangles), mesh.Triangles)
	}
	if err := mesh.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got := mesh.Area(); math.Abs(got-4) > 1e-12 {
		t.Fatalf("Area = %g, want 4", got)
	}
	if got := mesh.UsedVertices(); got != 5 {
		t.Fatalf("UsedVertices = %d, want 5", got)
	}
}

func TestTriangulateALattice(t *testing.T) {
	points := lattice(5)
	mesh, err := Triangulate(points)
	if err != nil {
		t.Fatalf("Triangulate: %v", err)
	}
	if err := mesh.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	// A triangulation of n points with b of them on the hull boundary has 2n-2-b
	// triangles, which for a five by five lattice is 32.
	if len(mesh.Triangles) != 32 {
		t.Fatalf("triangles = %d, want 32", len(mesh.Triangles))
	}
	if got := mesh.Area(); math.Abs(got-16) > 1e-12 {
		t.Fatalf("Area = %g, want 16", got)
	}
	if got := mesh.UsedVertices(); got != 25 {
		t.Fatalf("UsedVertices = %d, want 25", got)
	}
	if _, ok := mesh.IsDelaunay(); !ok {
		t.Fatalf("the mesh must be delaunay")
	}
}

func TestTriangulateCocircularPoints(t *testing.T) {
	// Every point lies on one circle, so every circumcircle test comes out exactly zero
	// and the algorithm has to keep going anyway.
	points := circle(12)
	mesh, err := Triangulate(points)
	if err != nil {
		t.Fatalf("Triangulate: %v", err)
	}
	if err := mesh.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(mesh.Triangles) != 10 {
		t.Fatalf("triangles = %d, want 10 for twelve points on a circle", len(mesh.Triangles))
	}
	if got := mesh.UsedVertices(); got != 12 {
		t.Fatalf("UsedVertices = %d, want 12", got)
	}
}

func TestTriangulateWithThreeCollinearPointsOnTheHull(t *testing.T) {
	points := []geom.Point{
		{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, {X: 1, Y: 1},
	}
	mesh, err := Triangulate(points)
	if err != nil {
		t.Fatalf("Triangulate: %v", err)
	}
	if err := mesh.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(mesh.Triangles) != 2 {
		t.Fatalf("triangles = %d, want 2: %v", len(mesh.Triangles), mesh.Triangles)
	}
	if got := mesh.Area(); math.Abs(got-1) > 1e-12 {
		t.Fatalf("Area = %g, want 1", got)
	}
}

func TestTriangulateOrdersItsTrianglesTheSameWayEveryTime(t *testing.T) {
	points := lattice(4)
	first, err := Triangulate(points)
	if err != nil {
		t.Fatalf("Triangulate: %v", err)
	}
	second, err := Triangulate(points)
	if err != nil {
		t.Fatalf("Triangulate: %v", err)
	}
	if len(first.Triangles) != len(second.Triangles) {
		t.Fatalf("two runs gave %d and %d triangles", len(first.Triangles), len(second.Triangles))
	}
	for index := range first.Triangles {
		if first.Triangles[index].Key() != second.Triangles[index].Key() {
			t.Fatalf("triangle %d differs between two runs: %s and %s",
				index, first.Triangles[index], second.Triangles[index])
		}
	}
}

func TestTriangulateRejectsBadInput(t *testing.T) {
	cases := []struct {
		label  string
		points []geom.Point
	}{
		{"too few points", []geom.Point{{X: 0}, {X: 1}}},
		{"a point that is not finite", []geom.Point{{X: 0}, {X: 1}, {Y: math.NaN()}}},
		{"a repeated point", []geom.Point{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}, {X: 1, Y: 0}}},
	}
	for _, item := range cases {
		if _, err := Triangulate(item.points); err == nil {
			t.Fatalf("%s: Triangulate = nil error, want a failure", item.label)
		}
	}
}

func TestCloneIsIndependent(t *testing.T) {
	mesh, err := Triangulate(lattice(3))
	if err != nil {
		t.Fatalf("Triangulate: %v", err)
	}
	clone := mesh.Clone()
	clone.Points[0] = geom.Point{X: 99, Y: 99}
	clone.Triangles[0] = Triangle{A: 0, B: 0, C: 0}
	if mesh.Points[0].X == 99 {
		t.Fatalf("Clone shares its points with the original")
	}
	if mesh.Triangles[0].B == 0 && mesh.Triangles[0].C == 0 {
		t.Fatalf("Clone shares its triangles with the original")
	}
}

func TestCorners(t *testing.T) {
	mesh, err := Triangulate(lattice(3))
	if err != nil {
		t.Fatalf("Triangulate: %v", err)
	}
	corners, err := mesh.Corners(0)
	if err != nil {
		t.Fatalf("Corners: %v", err)
	}
	if geom.Orientation(corners[0], corners[1], corners[2]) != exact.CounterClockwise {
		t.Fatalf("the corners of a triangle come out counterclockwise")
	}
	for _, index := range []int{-1, len(mesh.Triangles)} {
		if _, err := mesh.Corners(index); err == nil {
			t.Fatalf("Corners(%d) = nil error, want a failure", index)
		}
	}
}

func TestVerifyCatchesABrokenMesh(t *testing.T) {
	mesh, err := Triangulate(lattice(3))
	if err != nil {
		t.Fatalf("Triangulate: %v", err)
	}
	clockwise := mesh.Clone()
	clockwise.Triangles[0].B, clockwise.Triangles[0].C = clockwise.Triangles[0].C, clockwise.Triangles[0].B
	if err := clockwise.Verify(); err == nil {
		t.Fatalf("a clockwise triangle must be reported")
	}
	repeated := mesh.Clone()
	repeated.Triangles = append(repeated.Triangles, repeated.Triangles[0])
	if err := repeated.Verify(); err == nil {
		t.Fatalf("a repeated triangle must be reported")
	}
	degenerate := mesh.Clone()
	degenerate.Triangles[0].B = degenerate.Triangles[0].A
	if err := degenerate.Verify(); err == nil {
		t.Fatalf("a triangle that repeats a vertex must be reported")
	}
	outOfRange := mesh.Clone()
	outOfRange.Triangles[0].C = len(outOfRange.Points)
	if err := outOfRange.Verify(); err == nil {
		t.Fatalf("a triangle naming a missing point must be reported")
	}
	empty := &Mesh{Points: mesh.Points}
	if err := empty.Verify(); err == nil {
		t.Fatalf("a mesh with no triangles must be reported")
	}
}

func TestIsDelaunayCatchesAViolation(t *testing.T) {
	// Two triangles of a square split along the wrong diagonal are still a valid
	// triangulation, but not a delaunay one once a point sits inside a circumcircle.
	mesh := &Mesh{
		Points: []geom.Point{
			{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 1}, {X: 0, Y: 1}, {X: 2, Y: 0.5},
		},
		Triangles: []Triangle{{A: 0, B: 1, C: 2}, {A: 0, B: 2, C: 3}},
	}
	offender, ok := mesh.IsDelaunay()
	if ok {
		t.Fatalf("the point in the middle violates a circumcircle")
	}
	if offender < 0 || offender >= len(mesh.Triangles) {
		t.Fatalf("IsDelaunay named triangle %d", offender)
	}
	if err := mesh.Verify(); err == nil {
		t.Fatalf("Verify must report the violation as well")
	}
}

func TestDescribeAndLongestEdge(t *testing.T) {
	mesh, err := Triangulate(lattice(3))
	if err != nil {
		t.Fatalf("Triangulate: %v", err)
	}
	got := mesh.Describe()
	for _, fragment := range []string{"9 point(s)", "8 triangle(s)", "delaunay"} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("Describe = %q, which is missing %q", got, fragment)
		}
	}
	if strings.Contains(got, "NOT delaunay") {
		t.Fatalf("Describe = %q, want a delaunay mesh", got)
	}
	if longest := mesh.LongestEdge(); math.Abs(longest-math.Sqrt2) > 1e-12 {
		t.Fatalf("LongestEdge = %g, want the diagonal of a unit square", longest)
	}
}
