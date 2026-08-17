// Package geom holds the points, boxes and polygons the rest of the toolkit works on.
//
// Everything here that has to decide a sign asks the exact predicates rather than
// comparing a floating point determinant against zero, because a mesh built on
// inconsistent answers to those questions is not a mesh. Everything here that returns
// a slice returns one the caller owns.
package geom

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"Tessera/internal/exact"
)

// Point is a location in the plane.
type Point struct {
	X float64
	Y float64
}

// Add returns the sum of two points read as vectors.
func (p Point) Add(q Point) Point { return Point{X: p.X + q.X, Y: p.Y + q.Y} }

// Sub returns the difference of two points read as vectors.
func (p Point) Sub(q Point) Point { return Point{X: p.X - q.X, Y: p.Y - q.Y} }

// Scale returns the point scaled about the origin.
func (p Point) Scale(factor float64) Point {
	return Point{X: p.X * factor, Y: p.Y * factor}
}

// Dot returns the inner product of two points read as vectors.
func (p Point) Dot(q Point) float64 { return p.X*q.X + p.Y*q.Y }

// Cross returns the signed area of the parallelogram the two vectors span.
func (p Point) Cross(q Point) float64 { return p.X*q.Y - p.Y*q.X }

// Dist2 returns the squared distance between two points, which is what a comparison
// needs and what avoids a square root.
func (p Point) Dist2(q Point) float64 {
	dx, dy := p.X-q.X, p.Y-q.Y
	return dx*dx + dy*dy
}

// Dist returns the distance between two points.
func (p Point) Dist(q Point) float64 { return math.Sqrt(p.Dist2(q)) }

// IsFinite reports whether both coordinates are ordinary numbers.
func (p Point) IsFinite() bool {
	return !math.IsNaN(p.X) && !math.IsNaN(p.Y) && !math.IsInf(p.X, 0) && !math.IsInf(p.Y, 0)
}

// Equal reports whether two points are the same point.
func (p Point) Equal(q Point) bool { return p.X == q.X && p.Y == q.Y }

// Less orders points by x and then by y, which is the order a sweep needs.
func (p Point) Less(q Point) bool {
	if p.X != q.X {
		return p.X < q.X
	}
	return p.Y < q.Y
}

// String renders the point.
func (p Point) String() string {
	return fmt.Sprintf("(%.6g, %.6g)", p.X, p.Y)
}

// Parse reads a point list written as x,y pairs separated by semicolons, spaces or
// newlines.
func Parse(text string) ([]Point, error) {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == ';' || r == '\n' || r == '\r' || r == ' ' || r == '\t'
	})
	if len(fields) == 0 {
		return nil, fmt.Errorf("no points were given")
	}
	out := make([]Point, 0, len(fields))
	for index, field := range fields {
		parts := strings.Split(field, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("point %d is %q, which is not an x,y pair", index+1, field)
		}
		x, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return nil, fmt.Errorf("point %d has a bad x coordinate: %q", index+1, parts[0])
		}
		y, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("point %d has a bad y coordinate: %q", index+1, parts[1])
		}
		point := Point{X: x, Y: y}
		if !point.IsFinite() {
			return nil, fmt.Errorf("point %d is not finite", index+1)
		}
		out = append(out, point)
	}
	return out, nil
}

// Orientation reports whether c lies left of the line from a to b, using the exact
// predicate.
func Orientation(a, b, c Point) int {
	return exact.Orient2D(a.X, a.Y, b.X, b.Y, c.X, c.Y)
}

// Collinear reports whether three points lie on one line.
func Collinear(a, b, c Point) bool { return Orientation(a, b, c) == exact.Collinear }

// InCircle reports whether d lies inside the circle through a, b and c, which must be
// counterclockwise.
func InCircle(a, b, c, d Point) int {
	return exact.InCircle(a.X, a.Y, b.X, b.Y, c.X, c.Y, d.X, d.Y)
}

// Area2 returns twice the signed area of a triangle, positive when counterclockwise.
func Area2(a, b, c Point) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}

// TriangleArea returns the unsigned area of a triangle.
func TriangleArea(a, b, c Point) float64 { return math.Abs(Area2(a, b, c)) / 2 }

// Circumcenter returns the centre of the circle through three points, reporting false
// when they are collinear and there is no such circle.
func Circumcenter(a, b, c Point) (Point, bool) {
	if Collinear(a, b, c) {
		return Point{}, false
	}
	bx, by := b.X-a.X, b.Y-a.Y
	cx, cy := c.X-a.X, c.Y-a.Y
	d := 2 * (bx*cy - by*cx)
	if d == 0 {
		return Point{}, false
	}
	bl := bx*bx + by*by
	cl := cx*cx + cy*cy
	out := Point{
		X: a.X + (bl*cy-cl*by)/d,
		Y: a.Y + (cl*bx-bl*cx)/d,
	}
	if !out.IsFinite() {
		return Point{}, false
	}
	return out, true
}

// Circumradius returns the radius of the circle through three points.
func Circumradius(a, b, c Point) (float64, bool) {
	centre, ok := Circumcenter(a, b, c)
	if !ok {
		return 0, false
	}
	return centre.Dist(a), true
}

// Box is an axis aligned rectangle.
type Box struct {
	Min Point
	Max Point
}

// BoxOf returns the smallest box holding every point.
func BoxOf(points []Point) (Box, error) {
	if len(points) == 0 {
		return Box{}, fmt.Errorf("the box of no points is undefined")
	}
	out := Box{Min: points[0], Max: points[0]}
	for _, point := range points[1:] {
		out.Min.X = math.Min(out.Min.X, point.X)
		out.Min.Y = math.Min(out.Min.Y, point.Y)
		out.Max.X = math.Max(out.Max.X, point.X)
		out.Max.Y = math.Max(out.Max.Y, point.Y)
	}
	return out, nil
}

// Width and Height are the extent of the box.
func (b Box) Width() float64 { return b.Max.X - b.Min.X }

// Height is the vertical extent of the box.
func (b Box) Height() float64 { return b.Max.Y - b.Min.Y }

// Centre is the middle of the box.
func (b Box) Centre() Point {
	return Point{X: (b.Min.X + b.Max.X) / 2, Y: (b.Min.Y + b.Max.Y) / 2}
}

// Diagonal is the length of the box diagonal, which is the natural length scale of a
// point set.
func (b Box) Diagonal() float64 { return b.Min.Dist(b.Max) }

// Contains reports whether a point lies in the box, boundary included.
func (b Box) Contains(p Point) bool {
	return p.X >= b.Min.X && p.X <= b.Max.X && p.Y >= b.Min.Y && p.Y <= b.Max.Y
}

// Expand returns the box grown about its centre by the given factor.
func (b Box) Expand(factor float64) Box {
	centre := b.Centre()
	halfWidth := b.Width() / 2 * factor
	halfHeight := b.Height() / 2 * factor
	return Box{
		Min: Point{X: centre.X - halfWidth, Y: centre.Y - halfHeight},
		Max: Point{X: centre.X + halfWidth, Y: centre.Y + halfHeight},
	}
}

// Corners returns the four corners of the box counterclockwise.
func (b Box) Corners() []Point {
	return []Point{
		{X: b.Min.X, Y: b.Min.Y},
		{X: b.Max.X, Y: b.Min.Y},
		{X: b.Max.X, Y: b.Max.Y},
		{X: b.Min.X, Y: b.Max.Y},
	}
}

// String renders the box.
func (b Box) String() string {
	return fmt.Sprintf("[%.6g, %.6g] x [%.6g, %.6g]", b.Min.X, b.Max.X, b.Min.Y, b.Max.Y)
}

// SortedLexicographic returns the points ordered by x and then y. The input is left
// untouched, because the position of a point in the caller's list is what every index in
// a mesh refers to, so reordering it in place would send those indices elsewhere.
func SortedLexicographic(points []Point) []Point {
	out := append([]Point(nil), points...)
	sort.Slice(out, func(i, j int) bool { return out[i].Less(out[j]) })
	return out
}

// Deduplicate returns the distinct points in their first appearance order.
func Deduplicate(points []Point) []Point {
	seen := make(map[Point]bool, len(points))
	out := make([]Point, 0, len(points))
	for _, point := range points {
		if seen[point] {
			continue
		}
		seen[point] = true
		out = append(out, point)
	}
	return out
}

// FirstDuplicate returns the indices of the first repeated point, reporting false when
// every point is distinct.
func FirstDuplicate(points []Point) (int, int, bool) {
	seen := make(map[Point]int, len(points))
	for index, point := range points {
		if first, ok := seen[point]; ok {
			return first, index, true
		}
		seen[point] = index
	}
	return 0, 0, false
}

// AllCollinear reports whether every point lies on one line, which is the case that
// has no triangulation.
func AllCollinear(points []Point) bool {
	if len(points) < 3 {
		return true
	}
	first := points[0]
	second := first
	found := false
	for _, point := range points[1:] {
		if !point.Equal(first) {
			second = point
			found = true
			break
		}
	}
	if !found {
		return true
	}
	for _, point := range points {
		if !Collinear(first, second, point) {
			return false
		}
	}
	return true
}

// PolygonArea returns the signed area of a closed polygon, positive when the vertices
// run counterclockwise.
func PolygonArea(polygon []Point) float64 {
	if len(polygon) < 3 {
		return 0
	}
	total := 0.0
	for index := range polygon {
		current := polygon[index]
		next := polygon[(index+1)%len(polygon)]
		total += current.Cross(next)
	}
	return total / 2
}

// PolygonPerimeter returns the length of the closed boundary of a polygon.
func PolygonPerimeter(polygon []Point) float64 {
	if len(polygon) < 2 {
		return 0
	}
	total := 0.0
	for index := range polygon {
		total += polygon[index].Dist(polygon[(index+1)%len(polygon)])
	}
	return total
}

// PolygonCentroid returns the area centroid of a polygon, reporting false when the
// polygon has no area to speak of.
func PolygonCentroid(polygon []Point) (Point, bool) {
	area := PolygonArea(polygon)
	if area == 0 {
		return Point{}, false
	}
	var x, y float64
	for index := range polygon {
		current := polygon[index]
		next := polygon[(index+1)%len(polygon)]
		cross := current.Cross(next)
		x += (current.X + next.X) * cross
		y += (current.Y + next.Y) * cross
	}
	return Point{X: x / (6 * area), Y: y / (6 * area)}, true
}

// IsConvex reports whether a polygon given counterclockwise turns the same way at
// every vertex.
func IsConvex(polygon []Point) bool {
	if len(polygon) < 3 {
		return false
	}
	for index := range polygon {
		a := polygon[index]
		b := polygon[(index+1)%len(polygon)]
		c := polygon[(index+2)%len(polygon)]
		if Orientation(a, b, c) == exact.Clockwise {
			return false
		}
	}
	return true
}

// SortAround orders points by their angle about a centre, which is how the vertices of
// a cell are put in order once they are known.
func SortAround(centre Point, points []Point) []Point {
	out := append([]Point(nil), points...)
	sort.Slice(out, func(i, j int) bool {
		left := math.Atan2(out[i].Y-centre.Y, out[i].X-centre.X)
		right := math.Atan2(out[j].Y-centre.Y, out[j].X-centre.X)
		if left != right {
			return left < right
		}
		return out[i].Less(out[j])
	})
	return out
}

// ClipConvex returns the part of a convex polygon inside the box, by cutting it
// against each edge of the box in turn.
func ClipConvex(polygon []Point, box Box) []Point {
	if len(polygon) < 3 {
		return nil
	}
	current := append([]Point(nil), polygon...)
	type halfPlane struct {
		inside func(Point) bool
		cut    func(Point, Point) Point
	}
	planes := []halfPlane{
		{func(p Point) bool { return p.X >= box.Min.X },
			func(a, b Point) Point { return interpolateX(a, b, box.Min.X) }},
		{func(p Point) bool { return p.X <= box.Max.X },
			func(a, b Point) Point { return interpolateX(a, b, box.Max.X) }},
		{func(p Point) bool { return p.Y >= box.Min.Y },
			func(a, b Point) Point { return interpolateY(a, b, box.Min.Y) }},
		{func(p Point) bool { return p.Y <= box.Max.Y },
			func(a, b Point) Point { return interpolateY(a, b, box.Max.Y) }},
	}
	for _, plane := range planes {
		if len(current) == 0 {
			return nil
		}
		next := make([]Point, 0, len(current)+1)
		for index := range current {
			from := current[index]
			to := current[(index+1)%len(current)]
			fromInside := plane.inside(from)
			toInside := plane.inside(to)
			if fromInside {
				next = append(next, from)
			}
			if fromInside != toInside {
				next = append(next, plane.cut(from, to))
			}
		}
		current = next
	}
	if len(current) < 3 {
		return nil
	}
	return current
}

// interpolateX returns the point on the segment at a given x.
func interpolateX(a, b Point, x float64) Point {
	if a.X == b.X {
		return Point{X: x, Y: a.Y}
	}
	t := (x - a.X) / (b.X - a.X)
	return Point{X: x, Y: a.Y + t*(b.Y-a.Y)}
}

// interpolateY returns the point on the segment at a given y.
func interpolateY(a, b Point, y float64) Point {
	if a.Y == b.Y {
		return Point{X: a.X, Y: y}
	}
	t := (y - a.Y) / (b.Y - a.Y)
	return Point{X: a.X + t*(b.X-a.X), Y: y}
}
