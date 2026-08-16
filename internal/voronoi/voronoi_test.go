package voronoi

import (
	"math"
	"strings"
	"testing"

	"Tessera/internal/delaunay"
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

// diagramOf builds the diagram of a point set or fails the test.
func diagramOf(t *testing.T, points []geom.Point) (*delaunay.Mesh, *Diagram) {
	t.Helper()
	mesh, err := delaunay.Triangulate(points)
	if err != nil {
		t.Fatalf("Triangulate: %v", err)
	}
	diagram, err := Build(mesh)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return mesh, diagram
}

func TestBuildOfALattice(t *testing.T) {
	mesh, diagram := diagramOf(t, lattice(5))
	if len(diagram.Cells) != 25 {
		t.Fatalf("cells = %d, want one per site", len(diagram.Cells))
	}
	// Only the nine interior sites of a five by five lattice have a finite cell.
	if got := diagram.Bounded(); got != 9 {
		t.Fatalf("bounded cells = %d, want 9", got)
	}
	// Each of those cells is the unit square around its site.
	if got := diagram.TotalArea(); math.Abs(got-9) > 1e-12 {
		t.Fatalf("TotalArea = %g, want 9", got)
	}
	if got := diagram.Coverage(); math.Abs(got-9.0/16) > 1e-12 {
		t.Fatalf("Coverage = %g, want nine sixteenths", got)
	}
	if diagram.Mesh() != mesh {
		t.Fatalf("the diagram must carry the mesh it came from")
	}
	if diagram.Clipped {
		t.Fatalf("a fresh diagram is not clipped")
	}
}

func TestCellsOfTheMiddleSite(t *testing.T) {
	_, diagram := diagramOf(t, lattice(3))
	cell, ok := diagram.Cell(4)
	if !ok {
		t.Fatalf("the middle site has a cell")
	}
	if !cell.Bounded {
		t.Fatalf("the middle site of a three by three lattice is interior")
	}
	if got := cell.Area(); math.Abs(got-1) > 1e-12 {
		t.Fatalf("the middle cell has area %g, want 1", got)
	}
	if got := len(cell.Vertices); got < 4 {
		t.Fatalf("the middle cell has %d corner(s), want at least 4", got)
	}
	if _, ok := diagram.Cell(99); ok {
		t.Fatalf("a site that is not in the mesh has no cell")
	}
}

func TestUnboundedCellHasNoArea(t *testing.T) {
	_, diagram := diagramOf(t, lattice(3))
	cell, ok := diagram.Cell(0)
	if !ok {
		t.Fatalf("the corner site has a cell record")
	}
	if cell.Bounded {
		t.Fatalf("the corner site of a lattice is on the hull, so its cell is unbounded")
	}
	if got := cell.Area(); got != 0 {
		t.Fatalf("an unbounded cell reports area %g, want 0", got)
	}
}

func TestCellAreaChecksTheSite(t *testing.T) {
	_, diagram := diagramOf(t, lattice(3))
	area, err := diagram.CellArea(4)
	if err != nil {
		t.Fatalf("CellArea: %v", err)
	}
	if math.Abs(area-1) > 1e-12 {
		t.Fatalf("CellArea = %g, want 1", area)
	}
	for _, site := range []int{-1, 9, 100} {
		if _, err := diagram.CellArea(site); err == nil {
			t.Fatalf("CellArea(%d) = nil error, want a failure", site)
		}
	}
}

func TestBuildRejectsABadMesh(t *testing.T) {
	if _, err := Build(nil); err == nil {
		t.Fatalf("a diagram without a mesh must be reported")
	}
	if _, err := Build(&delaunay.Mesh{Points: lattice(3)}); err == nil {
		t.Fatalf("a mesh with no triangles must be reported")
	}
}

func TestClipKeepsTheBoundedCells(t *testing.T) {
	_, diagram := diagramOf(t, lattice(5))
	box, err := geom.BoxOf(lattice(5))
	if err != nil {
		t.Fatalf("BoxOf: %v", err)
	}
	clipped, err := diagram.Clip(box)
	if err != nil {
		t.Fatalf("Clip: %v", err)
	}
	if !clipped.Clipped {
		t.Fatalf("a clipped diagram says so")
	}
	if got := len(clipped.Cells); got != 9 {
		t.Fatalf("the clipped diagram has %d cell(s), want the 9 bounded ones", got)
	}
	if got := clipped.Bounded(); got != 9 {
		t.Fatalf("Bounded = %d, want 9", got)
	}
	// The nine unit cells sit well inside the bounding box of the lattice, so clipping
	// takes nothing away.
	if got := clipped.TotalArea(); math.Abs(got-9) > 1e-12 {
		t.Fatalf("TotalArea after clipping = %g, want 9", got)
	}
}

func TestClipToASmallBoxCutsTheCells(t *testing.T) {
	_, diagram := diagramOf(t, lattice(5))
	small := geom.Box{Min: geom.Point{X: 1, Y: 1}, Max: geom.Point{X: 2, Y: 2}}
	clipped, err := diagram.Clip(small)
	if err != nil {
		t.Fatalf("Clip: %v", err)
	}
	if got := clipped.TotalArea(); got > 1+1e-12 {
		t.Fatalf("the clipped area %g is larger than the box", got)
	}
	if got := clipped.TotalArea(); got <= 0 {
		t.Fatalf("the clipped area = %g, want something inside the box", got)
	}
	if len(clipped.Cells) == 0 {
		t.Fatalf("the box overlaps several cells")
	}
}

func TestClipRejectsAnEmptyBox(t *testing.T) {
	_, diagram := diagramOf(t, lattice(3))
	for _, box := range []geom.Box{
		{},
		{Min: geom.Point{X: 1, Y: 1}, Max: geom.Point{X: 1, Y: 2}},
		{Min: geom.Point{X: 1, Y: 1}, Max: geom.Point{X: 2, Y: 1}},
	} {
		if _, err := diagram.Clip(box); err == nil {
			t.Fatalf("Clip to %s = nil error, want a failure", box)
		}
	}
}

func TestDescribe(t *testing.T) {
	_, diagram := diagramOf(t, lattice(3))
	got := diagram.Describe()
	for _, fragment := range []string{"9 cell(s)", "9 site(s)", "1 bounded"} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("Describe = %q, which is missing %q", got, fragment)
		}
	}
	if strings.Contains(got, "clipped") {
		t.Fatalf("Describe = %q, want no mention of clipping", got)
	}
}
