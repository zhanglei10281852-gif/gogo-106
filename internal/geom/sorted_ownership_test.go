package geom

import (
	"reflect"
	"testing"
)

// TestSortedLexicographicLeavesTheInputAlone sorts a point list and then keeps using the
// list it was given, because the position of a point in the input is what every index in
// a mesh refers to.
func TestSortedLexicographicLeavesTheInputAlone(t *testing.T) {
	points := []Point{{X: 2, Y: 1}, {X: 1, Y: 9}, {X: 2, Y: 0}}
	before := append([]Point(nil), points...)

	sorted := SortedLexicographic(points)

	if !reflect.DeepEqual(points, before) {
		t.Fatalf("the input became %v, want %v", points, before)
	}
	want := []Point{{X: 1, Y: 9}, {X: 2, Y: 0}, {X: 2, Y: 1}}
	if !reflect.DeepEqual(sorted, want) {
		t.Fatalf("SortedLexicographic = %v, want %v", sorted, want)
	}
	sorted[0] = Point{X: 99, Y: 99}
	if !reflect.DeepEqual(points, before) {
		t.Fatalf("writing to the result changed the input to %v", points)
	}
}
