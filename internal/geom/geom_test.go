package geom

import (
	"math"
	"reflect"
	"testing"

	"Tessera/internal/exact"
)

// close fails the test unless two numbers agree to the given tolerance.
func close(t *testing.T, got, want, tolerance float64, label string) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Fatalf("%s = %g, want %g within %g", label, got, want, tolerance)
	}
}

func TestPointArithmetic(t *testing.T) {
	a := Point{X: 1, Y: 2}
	b := Point{X: 4, Y: 6}
	if got := a.Add(b); got != (Point{X: 5, Y: 8}) {
		t.Fatalf("Add = %s", got)
	}
	if got := b.Sub(a); got != (Point{X: 3, Y: 4}) {
		t.Fatalf("Sub = %s", got)
	}
	if got := a.Scale(3); got != (Point{X: 3, Y: 6}) {
		t.Fatalf("Scale = %s", got)
	}
	if got := a.Dot(b); got != 16 {
		t.Fatalf("Dot = %g, want 16", got)
	}
	if got := a.Cross(b); got != -2 {
		t.Fatalf("Cross = %g, want -2", got)
	}
	if got := a.Dist2(b); got != 25 {
		t.Fatalf("Dist2 = %g, want 25", got)
	}
	if got := a.Dist(b); got != 5 {
		t.Fatalf("Dist = %g, want 5", got)
	}
	if !a.Equal(Point{X: 1, Y: 2}) {
		t.Fatalf("Equal must hold for the same coordinates")
	}
	if a.Equal(b) {
		t.Fatalf("Equal must not hold for different points")
	}
}

func TestPointLessOrdersByXThenY(t *testing.T) {
	if !(Point{X: 1, Y: 9}).Less(Point{X: 2, Y: 0}) {
		t.Fatalf("x decides first")
	}
	if !(Point{X: 1, Y: 0}).Less(Point{X: 1, Y: 1}) {
		t.Fatalf("y breaks the tie")
	}
	if (Point{X: 1, Y: 1}).Less(Point{X: 1, Y: 1}) {
		t.Fatalf("a point is not less than itself")
	}
}

func TestIsFiniteAndString(t *testing.T) {
	if !(Point{X: 1, Y: 2}).IsFinite() {
		t.Fatalf("an ordinary point is finite")
	}
	for _, point := range []Point{{X: math.NaN()}, {Y: math.Inf(1)}, {X: math.Inf(-1)}} {
		if point.IsFinite() {
			t.Fatalf("%v must not be finite", point)
		}
	}
	if got, want := (Point{X: 1.5, Y: -2}).String(), "(1.5, -2)"; got != want {
		t.Fatalf("String = %q, want %q", got, want)
	}
}

func TestParse(t *testing.T) {
	points, err := Parse("0,0; 1,2 ;-3.5,4e1")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []Point{{X: 0, Y: 0}, {X: 1, Y: 2}, {X: -3.5, Y: 40}}
	if !reflect.DeepEqual(points, want) {
		t.Fatalf("Parse = %v, want %v", points, want)
	}
	newlines, err := Parse("1,1\n2,2\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(newlines) != 2 {
		t.Fatalf("Parse read %d point(s) from two lines", len(newlines))
	}
	for _, text := range []string{"", "   ", "1", "1,2,3", "a,2", "1,b", "1,NaN", "1,Inf"} {
		if _, err := Parse(text); err == nil {
			t.Fatalf("Parse(%q) = nil error, want a failure", text)
		}
	}
}

func TestOrientationAndCollinear(t *testing.T) {
	a := Point{X: 0, Y: 0}
	b := Point{X: 1, Y: 0}
	if got := Orientation(a, b, Point{X: 0, Y: 1}); got != exact.CounterClockwise {
		t.Fatalf("Orientation = %d, want a left turn", got)
	}
	if got := Orientation(a, b, Point{X: 0, Y: -1}); got != exact.Clockwise {
		t.Fatalf("Orientation = %d, want a right turn", got)
	}
	if !Collinear(a, b, Point{X: 5, Y: 0}) {
		t.Fatalf("three points on the x axis are collinear")
	}
	if Collinear(a, b, Point{X: 5, Y: 1}) {
		t.Fatalf("a point off the line is not collinear")
	}
}

func TestInCircleWrapper(t *testing.T) {
	a := Point{X: 1, Y: 0}
	b := Point{X: 0, Y: 1}
	c := Point{X: -1, Y: 0}
	if got := InCircle(a, b, c, Point{}); got != 1 {
		t.Fatalf("InCircle at the centre = %d, want 1", got)
	}
	if got := InCircle(a, b, c, Point{X: 3, Y: 3}); got != -1 {
		t.Fatalf("InCircle far away = %d, want -1", got)
	}
}

func TestArea(t *testing.T) {
	a := Point{X: 0, Y: 0}
	b := Point{X: 4, Y: 0}
	c := Point{X: 0, Y: 3}
	if got := Area2(a, b, c); got != 12 {
		t.Fatalf("Area2 = %g, want 12", got)
	}
	if got := Area2(a, c, b); got != -12 {
		t.Fatalf("Area2 of the reversed triangle = %g, want -12", got)
	}
	if got := TriangleArea(a, c, b); got != 6 {
		t.Fatalf("TriangleArea = %g, want 6", got)
	}
	if got := TriangleArea(a, b, Point{X: 8, Y: 0}); got != 0 {
		t.Fatalf("a flat triangle has area %g, want 0", got)
	}
}

func TestCircumcenter(t *testing.T) {
	centre, ok := Circumcenter(Point{X: 0, Y: 0}, Point{X: 2, Y: 0}, Point{X: 0, Y: 2})
	if !ok {
		t.Fatalf("three corners of a square have a circumcentre")
	}
	if centre != (Point{X: 1, Y: 1}) {
		t.Fatalf("Circumcenter = %s, want (1, 1)", centre)
	}
	radius, ok := Circumradius(Point{X: 0, Y: 0}, Point{X: 2, Y: 0}, Point{X: 0, Y: 2})
	if !ok {
		t.Fatalf("the same triangle has a circumradius")
	}
	close(t, radius, math.Sqrt2, 1e-15, "circumradius")
	if _, ok := Circumcenter(Point{X: 0, Y: 0}, Point{X: 1, Y: 1}, Point{X: 2, Y: 2}); ok {
		t.Fatalf("collinear points have no circumcentre")
	}
	if _, ok := Circumradius(Point{X: 0, Y: 0}, Point{X: 1, Y: 1}, Point{X: 2, Y: 2}); ok {
		t.Fatalf("collinear points have no circumradius")
	}
}

func TestBox(t *testing.T) {
	points := []Point{{X: 1, Y: 5}, {X: -2, Y: 3}, {X: 4, Y: -1}}
	box, err := BoxOf(points)
	if err != nil {
		t.Fatalf("BoxOf: %v", err)
	}
	if box.Min != (Point{X: -2, Y: -1}) || box.Max != (Point{X: 4, Y: 5}) {
		t.Fatalf("BoxOf = %s", box)
	}
	if got := box.Width(); got != 6 {
		t.Fatalf("Width = %g, want 6", got)
	}
	if got := box.Height(); got != 6 {
		t.Fatalf("Height = %g, want 6", got)
	}
	if got := box.Centre(); got != (Point{X: 1, Y: 2}) {
		t.Fatalf("Centre = %s", got)
	}
	close(t, box.Diagonal(), math.Sqrt(72), 1e-12, "Diagonal")
	if !box.Contains(Point{X: 0, Y: 0}) || !box.Contains(box.Min) || !box.Contains(box.Max) {
		t.Fatalf("the box must contain its own corners and the origin")
	}
	if box.Contains(Point{X: 5, Y: 0}) {
		t.Fatalf("the box must not contain a point outside it")
	}
	wider := box.Expand(2)
	if !wider.Contains(box.Min) || !wider.Contains(box.Max) {
		t.Fatalf("the expanded box must cover the original")
	}
	if got := wider.Width(); got != 12 {
		t.Fatalf("the expanded width = %g, want 12", got)
	}
	if got := len(box.Corners()); got != 4 {
		t.Fatalf("a box has %d corner(s), want 4", got)
	}
	if got := PolygonArea(box.Corners()); got != 36 {
		t.Fatalf("the corners enclose %g, want 36 counterclockwise", got)
	}
	if got, want := box.String(), "[-2, 4] x [-1, 5]"; got != want {
		t.Fatalf("String = %q, want %q", got, want)
	}
	if _, err := BoxOf(nil); err == nil {
		t.Fatalf("the box of no points must be reported")
	}
}

func TestSortedLexicographicReturnsSortedPoints(t *testing.T) {
	got := SortedLexicographic([]Point{{X: 2, Y: 1}, {X: 1, Y: 9}, {X: 2, Y: 0}})
	want := []Point{{X: 1, Y: 9}, {X: 2, Y: 0}, {X: 2, Y: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SortedLexicographic = %v, want %v", got, want)
	}
}

func TestDeduplicateKeepsFirstAppearanceOrder(t *testing.T) {
	got := Deduplicate([]Point{{X: 1}, {X: 2}, {X: 1}, {X: 3}, {X: 2}})
	want := []Point{{X: 1}, {X: 2}, {X: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Deduplicate = %v, want %v", got, want)
	}
}

func TestFirstDuplicate(t *testing.T) {
	first, second, ok := FirstDuplicate([]Point{{X: 1}, {X: 2}, {X: 1}})
	if !ok {
		t.Fatalf("the repeated point was not found")
	}
	if first != 0 || second != 2 {
		t.Fatalf("FirstDuplicate = %d and %d, want 0 and 2", first, second)
	}
	if _, _, ok := FirstDuplicate([]Point{{X: 1}, {X: 2}}); ok {
		t.Fatalf("distinct points have no duplicate")
	}
}

func TestAllCollinear(t *testing.T) {
	cases := []struct {
		label  string
		points []Point
		want   bool
	}{
		{"empty", nil, true},
		{"one point", []Point{{X: 1}}, true},
		{"two points", []Point{{X: 1}, {X: 2}}, true},
		{"a line", []Point{{X: 0}, {X: 1}, {X: 2}, {X: 9}}, true},
		{"repeated points", []Point{{X: 1, Y: 1}, {X: 1, Y: 1}, {X: 1, Y: 1}}, true},
		{"a triangle", []Point{{X: 0}, {X: 1}, {X: 0, Y: 1}}, false},
		{"a line with one point off it", []Point{{X: 0}, {X: 1}, {X: 2}, {X: 1, Y: 1}}, false},
	}
	for _, item := range cases {
		if got := AllCollinear(item.points); got != item.want {
			t.Fatalf("%s: AllCollinear = %t, want %t", item.label, got, item.want)
		}
	}
}

func TestPolygonMeasures(t *testing.T) {
	square := []Point{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}
	if got := PolygonArea(square); got != 4 {
		t.Fatalf("PolygonArea = %g, want 4", got)
	}
	reversed := []Point{{X: 0, Y: 2}, {X: 2, Y: 2}, {X: 2, Y: 0}, {X: 0, Y: 0}}
	if got := PolygonArea(reversed); got != -4 {
		t.Fatalf("PolygonArea of a clockwise square = %g, want -4", got)
	}
	if got := PolygonPerimeter(square); got != 8 {
		t.Fatalf("PolygonPerimeter = %g, want 8", got)
	}
	centroid, ok := PolygonCentroid(square)
	if !ok {
		t.Fatalf("a square has a centroid")
	}
	if centroid != (Point{X: 1, Y: 1}) {
		t.Fatalf("PolygonCentroid = %s, want (1, 1)", centroid)
	}
	if got := PolygonArea([]Point{{X: 0}, {X: 1}}); got != 0 {
		t.Fatalf("a segment encloses %g, want 0", got)
	}
	if got := PolygonPerimeter([]Point{{X: 0}}); got != 0 {
		t.Fatalf("a single point has perimeter %g, want 0", got)
	}
	if _, ok := PolygonCentroid([]Point{{X: 0}, {X: 1}, {X: 2}}); ok {
		t.Fatalf("a flat polygon has no centroid")
	}
}

func TestIsConvex(t *testing.T) {
	square := []Point{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}
	if !IsConvex(square) {
		t.Fatalf("a square is convex")
	}
	dart := []Point{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 4}}
	if IsConvex(dart) {
		t.Fatalf("a polygon with a reflex vertex is not convex")
	}
	if IsConvex([]Point{{X: 0}, {X: 1}}) {
		t.Fatalf("a segment is not a convex polygon")
	}
}

func TestSortAround(t *testing.T) {
	centre := Point{X: 0, Y: 0}
	points := []Point{{X: 0, Y: 1}, {X: 1, Y: 0}, {X: -1, Y: 0}, {X: 0, Y: -1}}
	got := SortAround(centre, points)
	if len(got) != 4 {
		t.Fatalf("SortAround returned %d point(s)", len(got))
	}
	if got[0] != (Point{X: 0, Y: -1}) {
		t.Fatalf("the walk starts at the smallest angle, got %s", got[0])
	}
	if PolygonArea(got) <= 0 {
		t.Fatalf("the sorted polygon must run counterclockwise, area %g", PolygonArea(got))
	}
	if !reflect.DeepEqual(points[0], Point{X: 0, Y: 1}) {
		t.Fatalf("SortAround modified its input")
	}
}

func TestClipConvex(t *testing.T) {
	box := Box{Min: Point{X: 0, Y: 0}, Max: Point{X: 2, Y: 2}}
	inside := []Point{{X: 0.5, Y: 0.5}, {X: 1.5, Y: 0.5}, {X: 1.5, Y: 1.5}, {X: 0.5, Y: 1.5}}
	if got := PolygonArea(ClipConvex(inside, box)); got != 1 {
		t.Fatalf("a polygon inside the box keeps its area, got %g", got)
	}
	overlapping := []Point{{X: 1, Y: 1}, {X: 5, Y: 1}, {X: 5, Y: 5}, {X: 1, Y: 5}}
	if got := PolygonArea(ClipConvex(overlapping, box)); got != 1 {
		t.Fatalf("the clipped area = %g, want 1", got)
	}
	outside := []Point{{X: 5, Y: 5}, {X: 6, Y: 5}, {X: 6, Y: 6}}
	if got := ClipConvex(outside, box); got != nil {
		t.Fatalf("a polygon outside the box clips to nothing, got %v", got)
	}
	if got := ClipConvex([]Point{{X: 0}, {X: 1}}, box); got != nil {
		t.Fatalf("a segment clips to nothing, got %v", got)
	}
	covering := []Point{{X: -5, Y: -5}, {X: 9, Y: -5}, {X: 9, Y: 9}, {X: -5, Y: 9}}
	if got := PolygonArea(ClipConvex(covering, box)); got != 4 {
		t.Fatalf("a polygon covering the box clips to the box, area %g want 4", got)
	}
}
