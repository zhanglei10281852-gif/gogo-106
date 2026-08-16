// Package delaunay triangulates a point set by incremental insertion.
//
// The method is the cavity form of the Bowyer and Watson algorithm, and it is built
// here without an enclosing triangle. The usual trick of starting from a huge triangle
// and deleting it afterwards fails on nearly collinear input: the slivers such input
// produces have circumcircles millions of times larger than the point set, those
// circles swallow the enclosing vertices, and at the end nothing is left. So the
// triangulation starts from three points of the set itself and grows.
//
// Inserting a point has two halves that are usually presented as different algorithms.
// If the point falls inside the current triangulation, every triangle whose circumcircle
// contains it is deleted and the cavity is filled with triangles from the point to each
// edge of the cavity boundary. If the point falls outside, the boundary edges it can see
// are joined to it as well. Both halves are the same statement: the region to retriangulate
// is bounded by the symmetric difference of the cavity edges and the visible boundary
// edges, and that region is star shaped about the new point.
//
// Two details decide whether the implementation works at all. The circumcircle test has
// to be exact, or the cavity can come out with a hole in it. And a new point that lies
// exactly on an edge of the boundary must not become a triangle with that edge, because
// such a triangle has no area and no business existing.
package delaunay

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"Tessera/internal/exact"
	"Tessera/internal/geom"
)

// Triangle names its three vertices by their index in the mesh, counterclockwise.
type Triangle struct {
	A int
	B int
	C int
}

// Vertices returns the three indices.
func (t Triangle) Vertices() [3]int { return [3]int{t.A, t.B, t.C} }

// Has reports whether the triangle uses a vertex.
func (t Triangle) Has(index int) bool { return t.A == index || t.B == index || t.C == index }

// Edges returns the three directed edges of the triangle.
func (t Triangle) Edges() [3][2]int {
	return [3][2]int{{t.A, t.B}, {t.B, t.C}, {t.C, t.A}}
}

// Key returns the vertices in increasing order, which identifies the triangle
// regardless of where its numbering starts.
func (t Triangle) Key() [3]int {
	out := t.Vertices()
	sort.Ints(out[:])
	return out
}

// String renders the triangle.
func (t Triangle) String() string { return fmt.Sprintf("(%d %d %d)", t.A, t.B, t.C) }

// Mesh is a triangulation of a point set.
type Mesh struct {
	Points    []geom.Point
	Triangles []Triangle
}

// Triangulate returns the Delaunay triangulation of the points. The indices of the
// result refer to the points as they were given.
func Triangulate(points []geom.Point) (*Mesh, error) {
	if len(points) < 3 {
		return nil, fmt.Errorf("a triangulation needs at least 3 points, got %d", len(points))
	}
	for index, point := range points {
		if !point.IsFinite() {
			return nil, fmt.Errorf("point %d is not finite", index+1)
		}
	}
	if first, second, repeated := geom.FirstDuplicate(points); repeated {
		return nil, fmt.Errorf("point %d repeats point %d", second+1, first+1)
	}
	if geom.AllCollinear(points) {
		return nil, fmt.Errorf("all %d points lie on one line, so they have no triangulation", len(points))
	}
	seed, err := seedTriangle(points)
	if err != nil {
		return nil, err
	}
	triangles := []Triangle{oriented(points, Triangle{A: seed[0], B: seed[1], C: seed[2]})}
	placed := map[int]bool{seed[0]: true, seed[1]: true, seed[2]: true}
	for index := range points {
		if placed[index] {
			continue
		}
		triangles, err = insert(points, triangles, index)
		if err != nil {
			return nil, err
		}
		placed[index] = true
	}
	out := &Mesh{
		Points:    append([]geom.Point(nil), points...),
		Triangles: triangles,
	}
	if len(out.Triangles) == 0 {
		return nil, fmt.Errorf("the insertion produced no triangles, so the mesh is empty")
	}
	sort.Slice(out.Triangles, func(i, j int) bool {
		left, right := out.Triangles[i].Key(), out.Triangles[j].Key()
		for position := 0; position < 3; position++ {
			if left[position] != right[position] {
				return left[position] < right[position]
			}
		}
		return false
	})
	return out, nil
}

// seedTriangle returns three points of the set that are not collinear, which is where
// the triangulation starts. The first three points of a lattice are collinear, so the
// search cannot simply take them.
func seedTriangle(points []geom.Point) ([3]int, error) {
	for third := 2; third < len(points); third++ {
		for second := 1; second < third; second++ {
			for first := 0; first < second; first++ {
				if !geom.Collinear(points[first], points[second], points[third]) {
					return [3]int{first, second, third}, nil
				}
			}
		}
	}
	return [3]int{}, fmt.Errorf("no three of the %d points form a triangle", len(points))
}

// insert adds one point to a triangulation and returns the new triangle list.
//
// The region that has to be retriangulated is bounded by the edges that appear once
// among the triangles whose circumcircle holds the point, together with the boundary
// edges the point can see from outside, and an edge in both sets is interior to the
// region rather than on its border.
func insert(points []geom.Point, triangles []Triangle, index int) ([]Triangle, error) {
	point := points[index]
	bad := make(map[int]bool, len(triangles))
	for position, triangle := range triangles {
		if inCircle(points, triangle, point) > 0 {
			bad[position] = true
		}
	}
	whole := make(map[[2]int]int, 3*len(triangles))
	for _, triangle := range triangles {
		for _, edge := range triangle.Edges() {
			whole[canonical(edge)]++
		}
	}
	cavity := make(map[[2]int]int, 3*len(bad))
	for position := range bad {
		for _, edge := range triangles[position].Edges() {
			cavity[canonical(edge)]++
		}
	}
	border := make([][2]int, 0, len(bad)+3)
	for position := range bad {
		for _, edge := range triangles[position].Edges() {
			key := canonical(edge)
			if cavity[key] != 1 {
				continue
			}
			if whole[key] == 1 && visible(points, edge, point) {
				continue
			}
			border = append(border, edge)
		}
	}
	for position, triangle := range triangles {
		if bad[position] {
			continue
		}
		for _, edge := range triangle.Edges() {
			if whole[canonical(edge)] != 1 {
				continue
			}
			if !visible(points, edge, point) {
				continue
			}
			border = append(border, edge)
		}
	}
	if len(border) == 0 {
		return nil, fmt.Errorf("point %d has no region to fill, so the mesh is inconsistent", index+1)
	}
	next := make([]Triangle, 0, len(triangles)+len(border))
	for position, triangle := range triangles {
		if !bad[position] {
			next = append(next, triangle)
		}
	}
	for _, edge := range border {
		// A point on the border splits that edge instead of forming a triangle with it,
		// and a triangle with no area would break every later orientation test.
		if geom.Orientation(points[edge[0]], points[edge[1]], point) == exact.Collinear {
			continue
		}
		next = append(next, oriented(points, Triangle{A: edge[0], B: edge[1], C: index}))
	}
	return next, nil
}

// visible reports whether a point lies strictly outside a boundary edge. The triangles
// are counterclockwise, so a boundary edge runs counterclockwise around the mesh and a
// point outside it lies to its right.
func visible(points []geom.Point, edge [2]int, point geom.Point) bool {
	return geom.Orientation(points[edge[0]], points[edge[1]], point) == exact.Clockwise
}

// canonical returns the edge with its endpoints in increasing order.
func canonical(edge [2]int) [2]int {
	if edge[0] <= edge[1] {
		return edge
	}
	return [2]int{edge[1], edge[0]}
}

// oriented returns the triangle with its vertices counterclockwise.
func oriented(points []geom.Point, triangle Triangle) Triangle {
	if geom.Orientation(points[triangle.A], points[triangle.B], points[triangle.C]) == exact.Clockwise {
		triangle.B, triangle.C = triangle.C, triangle.B
	}
	return triangle
}

// inCircle reports whether a point lies inside the circumcircle of a triangle.
func inCircle(points []geom.Point, triangle Triangle, point geom.Point) int {
	return geom.InCircle(points[triangle.A], points[triangle.B], points[triangle.C], point)
}

// Clone returns an independent copy of the mesh.
func (m *Mesh) Clone() *Mesh {
	return &Mesh{
		Points:    append([]geom.Point(nil), m.Points...),
		Triangles: append([]Triangle(nil), m.Triangles...),
	}
}

// Corners returns the three points of one triangle.
func (m *Mesh) Corners(index int) ([3]geom.Point, error) {
	if index < 0 || index >= len(m.Triangles) {
		return [3]geom.Point{}, fmt.Errorf("triangle %d is not in a mesh of %d", index, len(m.Triangles))
	}
	triangle := m.Triangles[index]
	return [3]geom.Point{m.Points[triangle.A], m.Points[triangle.B], m.Points[triangle.C]}, nil
}

// Area returns the total area of the triangles, which for a Delaunay triangulation is
// the area of the convex hull of the points.
func (m *Mesh) Area() float64 {
	total := 0.0
	for index := range m.Triangles {
		corners, err := m.Corners(index)
		if err != nil {
			continue
		}
		total += geom.TriangleArea(corners[0], corners[1], corners[2])
	}
	return total
}

// UsedVertices returns how many of the points appear in at least one triangle.
func (m *Mesh) UsedVertices() int {
	seen := make(map[int]bool, len(m.Points))
	for _, triangle := range m.Triangles {
		for _, vertex := range triangle.Vertices() {
			seen[vertex] = true
		}
	}
	return len(seen)
}

// IsDelaunay reports whether no point lies inside the circumcircle of any triangle,
// naming the first triangle that fails.
func (m *Mesh) IsDelaunay() (int, bool) {
	for index, triangle := range m.Triangles {
		for vertex, point := range m.Points {
			if triangle.Has(vertex) {
				continue
			}
			if inCircle(m.Points, triangle, point) > 0 {
				return index, false
			}
		}
	}
	return -1, true
}

// Verify checks the properties a triangulation of this kind must have: every triangle
// counterclockwise and with area, no triangle repeated, and the empty circumcircle
// property.
func (m *Mesh) Verify() error {
	if len(m.Triangles) == 0 {
		return fmt.Errorf("the mesh has no triangles")
	}
	seen := make(map[[3]int]bool, len(m.Triangles))
	for index, triangle := range m.Triangles {
		for _, vertex := range triangle.Vertices() {
			if vertex < 0 || vertex >= len(m.Points) {
				return fmt.Errorf("triangle %d names vertex %d, which is not a point", index, vertex)
			}
		}
		if triangle.A == triangle.B || triangle.B == triangle.C || triangle.A == triangle.C {
			return fmt.Errorf("triangle %d repeats a vertex: %s", index, triangle)
		}
		corners, err := m.Corners(index)
		if err != nil {
			return err
		}
		if geom.Orientation(corners[0], corners[1], corners[2]) != exact.CounterClockwise {
			return fmt.Errorf("triangle %d is not counterclockwise: %s", index, triangle)
		}
		key := triangle.Key()
		if seen[key] {
			return fmt.Errorf("triangle %s appears more than once", triangle)
		}
		seen[key] = true
	}
	if offender, ok := m.IsDelaunay(); !ok {
		return fmt.Errorf("triangle %d %s has a point inside its circumcircle",
			offender, m.Triangles[offender])
	}
	return nil
}

// Describe renders the mesh for a report.
func (m *Mesh) Describe() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%d point(s), %d triangle(s), area %.6g",
		len(m.Points), len(m.Triangles), m.Area())
	if _, ok := m.IsDelaunay(); ok {
		builder.WriteString(", delaunay")
		return builder.String()
	}
	builder.WriteString(", NOT delaunay")
	return builder.String()
}

// LongestEdge returns the length of the longest edge in the mesh, which is the scale a
// quality report is read against.
func (m *Mesh) LongestEdge() float64 {
	worst := 0.0
	for _, triangle := range m.Triangles {
		for _, edge := range triangle.Edges() {
			if length := m.Points[edge[0]].Dist(m.Points[edge[1]]); length > worst {
				worst = math.Max(worst, length)
			}
		}
	}
	return worst
}
