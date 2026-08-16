package topology

import (
	"reflect"
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

// meshOf triangulates a point set or fails the test.
func meshOf(t *testing.T, points []geom.Point) *delaunay.Mesh {
	t.Helper()
	mesh, err := delaunay.Triangulate(points)
	if err != nil {
		t.Fatalf("Triangulate: %v", err)
	}
	return mesh
}

func TestEdgeOf(t *testing.T) {
	if got := EdgeOf(5, 2); got != (Edge{Low: 2, High: 5}) {
		t.Fatalf("EdgeOf = %s, want 2-5", got)
	}
	if got := EdgeOf(2, 5); got != (Edge{Low: 2, High: 5}) {
		t.Fatalf("EdgeOf = %s, want 2-5", got)
	}
	if got, want := EdgeOf(2, 5).String(), "2-5"; got != want {
		t.Fatalf("String = %q, want %q", got, want)
	}
}

func TestCountsOfALattice(t *testing.T) {
	graph := Build(meshOf(t, lattice(5)))
	if got := graph.VertexCount(); got != 25 {
		t.Fatalf("VertexCount = %d, want 25", got)
	}
	if got := graph.TriangleCount(); got != 32 {
		t.Fatalf("TriangleCount = %d, want 32", got)
	}
	if got := graph.EdgeCount(); got != 56 {
		t.Fatalf("EdgeCount = %d, want 56", got)
	}
	if got := len(graph.InteriorEdges()); got != 40 {
		t.Fatalf("interior edges = %d, want 40", got)
	}
	if got := len(graph.OverusedEdges()); got != 0 {
		t.Fatalf("overused edges = %d, want 0", got)
	}
	if got := graph.Euler(); got != 2 {
		t.Fatalf("Euler = %d, want 2", got)
	}
	if !graph.Connected() {
		t.Fatalf("a lattice triangulation is one piece")
	}
}

func TestEdgesComeBackInOrder(t *testing.T) {
	graph := Build(meshOf(t, lattice(3)))
	edges := graph.Edges()
	if len(edges) != graph.EdgeCount() {
		t.Fatalf("Edges returned %d of %d edge(s)", len(edges), graph.EdgeCount())
	}
	for index := 1; index < len(edges); index++ {
		previous, current := edges[index-1], edges[index]
		if previous.Low > current.Low || (previous.Low == current.Low && previous.High >= current.High) {
			t.Fatalf("edge %s comes after %s", previous, current)
		}
	}
	for _, edge := range edges {
		users := graph.Uses(edge)
		if len(users) < 1 || len(users) > 2 {
			t.Fatalf("edge %s belongs to %d triangle(s)", edge, len(users))
		}
	}
	if got := graph.Uses(EdgeOf(0, 99)); len(got) != 0 {
		t.Fatalf("an edge that is not in the mesh has %d user(s)", len(got))
	}
}

func TestUsesReturnsACopy(t *testing.T) {
	graph := Build(meshOf(t, lattice(3)))
	edge := graph.Edges()[0]
	users := graph.Uses(edge)
	if len(users) == 0 {
		t.Fatalf("edge %s has no users", edge)
	}
	users[0] = 999
	if graph.Uses(edge)[0] == 999 {
		t.Fatalf("Uses handed out its own storage")
	}
}

func TestDegreesAndNeighbours(t *testing.T) {
	graph := Build(meshOf(t, lattice(3)))
	// The middle of a three by three lattice reaches its four orthogonal neighbours and
	// one diagonal of each of the four squares around it, which is six edges rather than
	// eight: only one diagonal of a square survives a triangulation.
	if got := graph.Degree(4); got != 6 {
		t.Fatalf("the degree of the middle vertex = %d, want 6", got)
	}
	if got := graph.MaxDegree(); got != 6 {
		t.Fatalf("MaxDegree = %d, want 6", got)
	}
	neighbours := graph.Neighbours(4)
	if len(neighbours) != 6 {
		t.Fatalf("Neighbours(4) = %v, want six of them", neighbours)
	}
	for index, vertex := range neighbours {
		if vertex < 0 || vertex > 8 || vertex == 4 {
			t.Fatalf("Neighbours(4) holds %d, which is not another vertex", vertex)
		}
		if index > 0 && vertex <= neighbours[index-1] {
			t.Fatalf("Neighbours came back out of order: %v", neighbours)
		}
	}
	// A corner of the lattice reaches its two sides, and the diagonal of its square only
	// if that diagonal was the one the triangulation chose.
	corner := graph.Neighbours(0)
	if len(corner) < 2 || len(corner) > 3 {
		t.Fatalf("Neighbours(0) = %v, want two or three of them", corner)
	}
	if !reflect.DeepEqual(corner[:2], []int{1, 3}) {
		t.Fatalf("Neighbours(0) = %v, want the two sides first", corner)
	}
	neighbours[0] = 999
	if graph.Neighbours(4)[0] == 999 {
		t.Fatalf("Neighbours handed out its own storage")
	}
	if got := graph.Degree(99); got != 0 {
		t.Fatalf("a vertex that is not in the mesh has degree %d", got)
	}
	triangles := graph.IncidentTriangles(4)
	if len(triangles) != 6 {
		t.Fatalf("the middle vertex is in %d triangle(s), want 6", len(triangles))
	}
	for index := 1; index < len(triangles); index++ {
		if triangles[index] <= triangles[index-1] {
			t.Fatalf("IncidentTriangles came back out of order: %v", triangles)
		}
	}
	if got := graph.IncidentTriangles(99); len(got) != 0 {
		t.Fatalf("a vertex that is not in the mesh is in %d triangle(s)", len(got))
	}
}

func TestEveryTriangleIsReachable(t *testing.T) {
	for _, side := range []int{3, 4, 5} {
		graph := Build(meshOf(t, lattice(side)))
		if !graph.Connected() {
			t.Fatalf("a %d by %d lattice triangulation is one piece", side, side)
		}
	}
	empty := Build(&delaunay.Mesh{Points: lattice(3)})
	if empty.Connected() {
		t.Fatalf("a mesh with no triangles is not connected")
	}
}

func TestEulerOfSeveralPointSets(t *testing.T) {
	cases := [][]geom.Point{
		{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}, {X: 1, Y: 1}},
		lattice(3),
		lattice(4),
		{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, {X: 1, Y: 1}},
	}
	for index, points := range cases {
		graph := Build(meshOf(t, points))
		if got := graph.Euler(); got != 2 {
			t.Fatalf("point set %d has euler characteristic %d, want 2", index, got)
		}
	}
}

func TestDescribe(t *testing.T) {
	graph := Build(meshOf(t, lattice(3)))
	got := graph.Describe()
	for _, fragment := range []string{"9 vertex(es)", "16 edge(s)", "8 triangle(s)", "euler 2"} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("Describe = %q, which is missing %q", got, fragment)
		}
	}
}
