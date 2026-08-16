package report

import (
	"math"
	"testing"
)

func TestTableRender(t *testing.T) {
	table := Table{
		Header: []string{"a", "bb"},
		Rows:   [][]string{{"ccc", "d"}},
		Gap:    2,
	}
	want := "a    bb\n---  --\nccc  d\n"
	if got := table.Render(); got != want {
		t.Fatalf("Render = %q, want %q", got, want)
	}
}

func TestTableAlignsRightAndTrimsPadding(t *testing.T) {
	table := Table{
		Rows:       [][]string{{"ccc", "d"}, {"e", "ffff"}},
		Alignments: []Alignment{AlignLeft, AlignRight},
	}
	want := "ccc     d\ne    ffff\n"
	if got := table.Render(); got != want {
		t.Fatalf("Render = %q, want %q", got, want)
	}
}

func TestTableAddRowAndShortRows(t *testing.T) {
	table := Table{Gap: 1}
	table.AddRow("a", "b")
	table.AddRow("c")
	want := "a b\nc\n"
	if got := table.Render(); got != want {
		t.Fatalf("Render = %q, want %q", got, want)
	}
}

func TestTableCountsRunesNotBytes(t *testing.T) {
	table := Table{Rows: [][]string{{"日本語", "x"}, {"ab", "y"}}}
	want := "日本語  x\nab   y\n"
	if got := table.Render(); got != want {
		t.Fatalf("Render = %q, want %q", got, want)
	}
}

func TestFloat(t *testing.T) {
	cases := map[float64]string{
		0:                "0",
		1.5:              "1.5",
		1.0 / 3:          "0.333333",
		math.Inf(1):      "+inf",
		math.Inf(-1):     "-inf",
		1234567890123456: "1.23457e+15",
	}
	for value, want := range cases {
		if got := Float(value); got != want {
			t.Fatalf("Float(%v) = %q, want %q", value, got, want)
		}
	}
	if got := Float(math.NaN()); got != "nan" {
		t.Fatalf("Float(NaN) = %q, want nan", got)
	}
}

func TestSci(t *testing.T) {
	cases := map[float64]string{
		0:       "0",
		0.00123: "1.230e-03",
		-45678:  "-4.568e+04",
	}
	for value, want := range cases {
		if got := Sci(value); got != want {
			t.Fatalf("Sci(%v) = %q, want %q", value, got, want)
		}
	}
	if got := Sci(math.NaN()); got != "nan" {
		t.Fatalf("Sci(NaN) = %q, want nan", got)
	}
}

func TestBoolIntAndInts(t *testing.T) {
	if got := Bool(true); got != "yes" {
		t.Fatalf("Bool(true) = %q", got)
	}
	if got := Bool(false); got != "no" {
		t.Fatalf("Bool(false) = %q", got)
	}
	if got := Int(-7); got != "-7" {
		t.Fatalf("Int = %q", got)
	}
	if got, want := Ints([]int{2, 3, 5}), "2 3 5"; got != want {
		t.Fatalf("Ints = %q, want %q", got, want)
	}
	if got := Ints(nil); got != "" {
		t.Fatalf("Ints(nil) = %q, want the empty string", got)
	}
}

func TestBar(t *testing.T) {
	cases := []struct {
		count, largest, width int
		want                  string
	}{
		{10, 10, 4, "####"},
		{5, 10, 4, "##"},
		{1, 100, 10, "#"},
		{0, 10, 4, ""},
		{5, 0, 4, ""},
		{5, 10, 0, ""},
	}
	for _, item := range cases {
		if got := Bar(item.count, item.largest, item.width); got != item.want {
			t.Fatalf("Bar(%d, %d, %d) = %q, want %q",
				item.count, item.largest, item.width, got, item.want)
		}
	}
}
