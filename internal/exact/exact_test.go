package exact

import "testing"

func TestOrient2DOnWellSeparatedPoints(t *testing.T) {
	cases := []struct {
		label                  string
		ax, ay, bx, by, cx, cy float64
		want                   int
	}{
		{"left turn", 0, 0, 1, 0, 0, 1, CounterClockwise},
		{"right turn", 0, 0, 1, 0, 0, -1, Clockwise},
		{"straight along x", 0, 0, 1, 0, 2, 0, Collinear},
		{"straight along the diagonal", 0, 0, 1, 1, 2, 2, Collinear},
		{"far apart", -1000, -1000, 1000, 0, 0, 1000, CounterClockwise},
		{"repeated first point", 3, 4, 3, 4, 9, 9, Collinear},
	}
	for _, item := range cases {
		if got := Orient2D(item.ax, item.ay, item.bx, item.by, item.cx, item.cy); got != item.want {
			t.Fatalf("%s: Orient2D = %d, want %d", item.label, got, item.want)
		}
	}
}

func TestOrient2DFlipsWithTheOrderOfItsArguments(t *testing.T) {
	forward := Orient2D(0, 0, 4, 1, 1, 3)
	backward := Orient2D(4, 1, 0, 0, 1, 3)
	if forward != CounterClockwise {
		t.Fatalf("Orient2D = %d, want a left turn", forward)
	}
	if backward != -forward {
		t.Fatalf("swapping the first two points gave %d, want %d", backward, -forward)
	}
}

func TestOrient2DAgreesWithTheExactPathOnCleanInput(t *testing.T) {
	cases := [][6]float64{
		{0, 0, 1, 0, 0, 1},
		{0, 0, 1, 0, 0, -1},
		{-3, 2, 5, 7, 11, 13},
		{1.5, 2.5, -4.25, 8.125, 0, 0},
		{0, 0, 2, 2, 5, 5},
	}
	for _, item := range cases {
		filtered := Orient2D(item[0], item[1], item[2], item[3], item[4], item[5])
		truth := ExactOrient2D(item[0], item[1], item[2], item[3], item[4], item[5])
		if filtered != truth {
			t.Fatalf("the filter said %d and the exact path said %d for %v", filtered, truth, item)
		}
	}
}

func TestNaiveAgreesOnCleanInput(t *testing.T) {
	// Plain arithmetic is only wrong when the determinant nearly cancels, and none of
	// these do.
	cases := [][6]float64{
		{0, 0, 1, 0, 0, 1},
		{0, 0, 1, 0, 2, 0},
		{-3, 2, 5, 7, 11, 13},
	}
	for _, item := range cases {
		naive := NaiveOrient2D(item[0], item[1], item[2], item[3], item[4], item[5])
		truth := ExactOrient2D(item[0], item[1], item[2], item[3], item[4], item[5])
		if naive != truth {
			t.Fatalf("plain arithmetic said %d and the exact path said %d for %v", naive, truth, item)
		}
	}
}

func TestInCircle(t *testing.T) {
	// The unit circle through three of its points, taken counterclockwise.
	ax, ay := 1.0, 0.0
	bx, by := 0.0, 1.0
	cx, cy := -1.0, 0.0
	cases := []struct {
		label  string
		dx, dy float64
		want   int
	}{
		{"the centre", 0, 0, 1},
		{"just inside", 0.5, 0.5, 1},
		{"far outside", 5, 5, -1},
		{"on the circle", 0, -1, 0},
		{"just outside", 0.999, 0.999, -1},
	}
	for _, item := range cases {
		got := InCircle(ax, ay, bx, by, cx, cy, item.dx, item.dy)
		if got != item.want {
			t.Fatalf("%s: InCircle = %d, want %d", item.label, got, item.want)
		}
	}
}

func TestInCircleAgreesWithTheExactPath(t *testing.T) {
	cases := [][8]float64{
		{0, 0, 4, 0, 4, 4, 1, 1},
		{0, 0, 4, 0, 4, 4, 9, 9},
		{0, 0, 1, 0, 1, 1, 0, 1},
		{-2, -2, 6, -1, 3, 7, 0, 0},
	}
	for _, item := range cases {
		filtered := InCircle(item[0], item[1], item[2], item[3], item[4], item[5], item[6], item[7])
		truth := ExactInCircle(item[0], item[1], item[2], item[3], item[4], item[5], item[6], item[7])
		if filtered != truth {
			t.Fatalf("the filter said %d and the exact path said %d for %v", filtered, truth, item)
		}
	}
}

func TestInCircleOfASquareCorner(t *testing.T) {
	// The fourth corner of a square lies exactly on the circle through the other three.
	if got := InCircle(0, 0, 1, 0, 1, 1, 0, 1); got != 0 {
		t.Fatalf("InCircle = %d, want 0 for a cocircular point", got)
	}
}

func TestStatsCountEveryDecision(t *testing.T) {
	var predicates Predicates
	if got := predicates.Stats().Total(); got != 0 {
		t.Fatalf("a fresh counter has made %d decision(s), want 0", got)
	}
	calls := 0
	for _, item := range [][6]float64{
		{0, 0, 1, 0, 0, 1},
		{0, 0, 1, 1, 2, 2},
		{5, 5, -3, 2, 7, 11},
	} {
		predicates.Orient2D(item[0], item[1], item[2], item[3], item[4], item[5])
		calls++
	}
	predicates.InCircle(0, 0, 4, 0, 4, 4, 1, 1)
	calls++
	stats := predicates.Stats()
	if stats.Total() != calls {
		t.Fatalf("the counters add up to %d, want %d", stats.Total(), calls)
	}
	if stats.Filtered < 0 || stats.Exact < 0 {
		t.Fatalf("the counters went negative: %+v", stats)
	}
	predicates.Reset()
	if got := predicates.Stats().Total(); got != 0 {
		t.Fatalf("Reset left %d decision(s) behind", got)
	}
}

func TestNilPredicatesWork(t *testing.T) {
	var predicates *Predicates
	if got := predicates.Orient2D(0, 0, 1, 0, 0, 1); got != CounterClockwise {
		t.Fatalf("Orient2D on a nil counter = %d, want a left turn", got)
	}
	if got := predicates.InCircle(0, 0, 4, 0, 4, 4, 1, 1); got != 1 {
		t.Fatalf("InCircle on a nil counter = %d, want 1", got)
	}
	if got := predicates.Stats(); got.Total() != 0 {
		t.Fatalf("a nil counter reported %d decision(s)", got.Total())
	}
	predicates.Reset()
}
