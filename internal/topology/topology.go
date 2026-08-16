// Package topology reads the connectivity of a mesh.
//
// A triangulation is two things at once: a set of triangles with coordinates, and a
// graph. The graph is what answers the questions that have nothing to do with where
// the points are, and it is where the arithmetic-free consistency checks live. An edge
// that belongs to two triangles is interior; an edge that belongs to one is on the
// boundary; an edge that belongs to three means the mesh is broken. Those counts have
// to agree with Euler's formula, and if they do not, something above them is wrong.
package topology

import (
	"fmt"
	"sort"
	"strings"

	"Tessera/internal/delaunay"
	"Tessera/internal/geom"
)

// Edge is an undirected edge named by its two endpoints in increasing order.
type Edge struct {
	Low  int
	High int
}

// EdgeOf returns the edge between two vertices.
func EdgeOf(a, b int) Edge {
	if a <= b {
		return Edge{Low: a, High: b}
	}
	return Edge{Low: b, High: a}
}

// String renders the edge.
func (e Edge) String() string { return fmt.Sprintf("%d-%d", e.Low, e.High) }

// Graph is the connectivity of a mesh.
type Graph struct {
	mesh      *delaunay.Mesh
	uses      map[Edge][]int
	incident  map[int][]int
	neighbour map[int][]int
}

// Build reads the connectivity of a mesh.
func Build(mesh *delaunay.Mesh) *Graph {
	out := &Graph{
		mesh:      mesh,
		uses:      map[Edge][]int{},
		incident:  map[int][]int{},
		neighbour: map[int][]int{},
	}
	adjacent := map[int]map[int]bool{}
	for index, triangle := range mesh.Triangles {
		for _, vertex := range triangle.Vertices() {
			out.incident[vertex] = append(out.incident[vertex], index)
		}
		for _, edge := range triangle.Edges() {
			key := EdgeOf(edge[0], edge[1])
			out.uses[key] = append(out.uses[key], index)
			for _, pair := range [2][2]int{{edge[0], edge[1]}, {edge[1], edge[0]}} {
				if adjacent[pair[0]] == nil {
					adjacent[pair[0]] = map[int]bool{}
				}
				adjacent[pair[0]][pair[1]] = true
			}
		}
	}
	for vertex, set := range adjacent {
		for other := range set {
			out.neighbour[vertex] = append(out.neighbour[vertex], other)
		}
		sort.Ints(out.neighbour[vertex])
	}
	return out
}

// VertexCount is how many vertices appear in at least one triangle.
func (g *Graph) VertexCount() int { return len(g.incident) }

// TriangleCount is how many triangles the mesh has.
func (g *Graph) TriangleCount() int { return len(g.mesh.Triangles) }

// EdgeCount is how many distinct edges the mesh has.
func (g *Graph) EdgeCount() int { return len(g.uses) }

// Edges returns every edge in a stable order.
func (g *Graph) Edges() []Edge {
	out := make([]Edge, 0, len(g.uses))
	for edge := range g.uses {
		out = append(out, edge)
	}
	sortEdges(out)
	return out
}

// sortEdges puts edges in increasing order.
func sortEdges(edges []Edge) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Low != edges[j].Low {
			return edges[i].Low < edges[j].Low
		}
		return edges[i].High < edges[j].High
	})
}

// Uses returns the triangles that use an edge.
func (g *Graph) Uses(edge Edge) []int {
	return append([]int(nil), g.uses[edge]...)
}

// BoundaryEdges returns the edges that belong to exactly one triangle. For a Delaunay
// triangulation these are the edges of the convex hull, and any other count means the
// mesh does not describe a surface.
func (g *Graph) BoundaryEdges() []Edge {
	out := make([]Edge, 0, len(g.uses))
	for edge, users := range g.uses {
		if len(users) == 1 {
			out = append(out, edge)
		}
	}
	sortEdges(out)
	return out
}

// InteriorEdges returns the edges that belong to two triangles.
func (g *Graph) InteriorEdges() []Edge {
	out := make([]Edge, 0, len(g.uses))
	for edge, users := range g.uses {
		if len(users) == 2 {
			out = append(out, edge)
		}
	}
	sortEdges(out)
	return out
}

// OverusedEdges returns the edges that belong to three or more triangles, which should
// never happen.
func (g *Graph) OverusedEdges() []Edge {
	out := make([]Edge, 0)
	for edge, users := range g.uses {
		if len(users) > 2 {
			out = append(out, edge)
		}
	}
	sortEdges(out)
	return out
}

// BoundaryVertices returns the vertices on the boundary, in increasing order.
func (g *Graph) BoundaryVertices() []int {
	seen := map[int]bool{}
	for _, edge := range g.BoundaryEdges() {
		seen[edge.Low] = true
		seen[edge.High] = true
	}
	out := make([]int, 0, len(seen))
	for vertex := range seen {
		out = append(out, vertex)
	}
	sort.Ints(out)
	return out
}

// IsBoundaryVertex reports whether a vertex lies on the boundary.
func (g *Graph) IsBoundaryVertex(vertex int) bool {
	for _, edge := range g.BoundaryEdges() {
		if edge.Low == vertex || edge.High == vertex {
			return true
		}
	}
	return false
}

// BoundaryLength returns the total length of the boundary edges.
func (g *Graph) BoundaryLength() float64 {
	total := 0.0
	for _, edge := range g.BoundaryEdges() {
		total += g.mesh.Points[edge.Low].Dist(g.mesh.Points[edge.High])
	}
	return total
}

// Degree returns how many edges meet at a vertex.
func (g *Graph) Degree(vertex int) int { return len(g.neighbour[vertex]) }

// Neighbours returns the vertices joined to a vertex by an edge.
func (g *Graph) Neighbours(vertex int) []int {
	return append([]int(nil), g.neighbour[vertex]...)
}

// IncidentTriangles returns the triangles that use a vertex, in mesh order.
func (g *Graph) IncidentTriangles(vertex int) []int {
	out := append([]int(nil), g.incident[vertex]...)
	sort.Ints(out)
	return out
}

// MaxDegree returns the largest number of edges at any vertex.
func (g *Graph) MaxDegree() int {
	worst := 0
	for vertex := range g.incident {
		if degree := g.Degree(vertex); degree > worst {
			worst = degree
		}
	}
	return worst
}

// Euler returns the Euler characteristic counting the outer face, which is two for a
// triangulated disc.
func (g *Graph) Euler() int {
	return g.VertexCount() - g.EdgeCount() + g.TriangleCount() + 1
}

// Connected reports whether the triangles form one piece.
func (g *Graph) Connected() bool {
	if g.TriangleCount() == 0 {
		return false
	}
	seen := map[int]bool{0: true}
	queue := []int{0}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, edge := range g.mesh.Triangles[current].Edges() {
			for _, user := range g.uses[EdgeOf(edge[0], edge[1])] {
				if seen[user] {
					continue
				}
				seen[user] = true
				queue = append(queue, user)
			}
		}
	}
	return len(seen) == g.TriangleCount()
}

// Consistent checks the counting identity every triangulated surface obeys: each
// triangle has three edges, and every edge is used once if it is on the boundary and
// twice if it is not.
func (g *Graph) Consistent() error {
	if overused := g.OverusedEdges(); len(overused) > 0 {
		return fmt.Errorf("edge %s belongs to %d triangles", overused[0], len(g.Uses(overused[0])))
	}
	interior := len(g.InteriorEdges())
	boundary := len(g.BoundaryEdges())
	if interior+boundary != g.EdgeCount() {
		return fmt.Errorf("%d interior and %d boundary edges do not add up to %d edges",
			interior, boundary, g.EdgeCount())
	}
	if 2*interior+boundary != 3*g.TriangleCount() {
		return fmt.Errorf(
			"twice the %d interior edges plus the %d boundary edges is %d, but %d triangles have %d edge slots",
			interior, boundary, 2*interior+boundary, g.TriangleCount(), 3*g.TriangleCount())
	}
	if got := g.Euler(); got != 2 {
		return fmt.Errorf("the euler characteristic is %d, want 2", got)
	}
	return nil
}

// BoundaryPolygon returns the boundary vertices in the order they are walked, which is
// the convex hull of the points for a Delaunay triangulation.
func (g *Graph) BoundaryPolygon() ([]geom.Point, error) {
	edges := g.BoundaryEdges()
	if len(edges) == 0 {
		return nil, fmt.Errorf("the mesh has no boundary")
	}
	next := map[int][]int{}
	for _, edge := range edges {
		next[edge.Low] = append(next[edge.Low], edge.High)
		next[edge.High] = append(next[edge.High], edge.Low)
	}
	for vertex, ends := range next {
		if len(ends) != 2 {
			return nil, fmt.Errorf("boundary vertex %d has %d boundary neighbour(s), want 2",
				vertex, len(ends))
		}
	}
	start := edges[0].Low
	out := []geom.Point{g.mesh.Points[start]}
	previous := start
	current := next[start][0]
	for current != start {
		out = append(out, g.mesh.Points[current])
		ends := next[current]
		if ends[0] == previous {
			previous, current = current, ends[1]
			continue
		}
		previous, current = current, ends[0]
	}
	if len(out) != len(edges) {
		return nil, fmt.Errorf("the boundary walk visited %d vertex(es) over %d edge(s)",
			len(out), len(edges))
	}
	return out, nil
}

// Describe renders the connectivity for a report.
func (g *Graph) Describe() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%d vertex(es), %d edge(s), %d triangle(s)",
		g.VertexCount(), g.EdgeCount(), g.TriangleCount())
	fmt.Fprintf(&builder, ", %d on the boundary", len(g.BoundaryEdges()))
	fmt.Fprintf(&builder, ", euler %d", g.Euler())
	return builder.String()
}
