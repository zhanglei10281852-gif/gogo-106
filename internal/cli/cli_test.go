package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// run executes one invocation and returns the exit code with both streams.
func run(args ...string) (int, string, string) {
	var stdout, stderr strings.Builder
	code := Run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// wants fails the test unless the text holds every fragment.
func wants(t *testing.T, text string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			t.Fatalf("output is missing %q:\n%s", fragment, text)
		}
	}
}

func TestRunWithoutArgumentsPrintsUsage(t *testing.T) {
	code, stdout, stderr := run()
	if code != ExitError {
		t.Fatalf("code = %d, want %d", code, ExitError)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want nothing", stdout)
	}
	wants(t, stderr, "usage: tessera <command>")
}

func TestRunHelp(t *testing.T) {
	for _, argument := range []string{"help", "--help", "-h"} {
		code, stdout, _ := run(argument)
		if code != ExitOK {
			t.Fatalf("%s: code = %d, want %d", argument, code, ExitOK)
		}
		wants(t, stdout,
			"usage: tessera <command>",
			"study tool only",
			"exit codes: 0 produced an answer, 1 could not run, 2 ran and found nothing",
			"sample", "hull", "mesh", "voronoi", "quality", "topology", "predicates", "report",
			"point sets: circle, clustered, grid, line, nearline, spiral")
	}
}

func TestRunVersion(t *testing.T) {
	for _, argument := range []string{"version", "--version", "-v"} {
		code, stdout, _ := run(argument)
		if code != ExitOK {
			t.Fatalf("%s: code = %d, want %d", argument, code, ExitOK)
		}
		wants(t, stdout, Version)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	code, _, stderr := run("mush")
	if code != ExitError {
		t.Fatalf("code = %d, want %d", code, ExitError)
	}
	wants(t, stderr, `unknown command "mush"`, "usage: tessera <command>")
}

func TestSubcommandHelpFlag(t *testing.T) {
	code, stdout, _ := run("mesh", "-h")
	if code != ExitOK {
		t.Fatalf("code = %d, want %d", code, ExitOK)
	}
	wants(t, stdout, "mesh - triangulate a point set", "-kind", "-n", "-points", "-show")
}

func TestSubcommandRejectsAnUnknownFlag(t *testing.T) {
	code, _, stderr := run("mesh", "-nope")
	if code != ExitError {
		t.Fatalf("code = %d, want %d", code, ExitError)
	}
	wants(t, stderr, "flag provided but not defined")
}

func TestSample(t *testing.T) {
	code, stdout, stderr := run("sample", "-kind", "grid", "-n", "9")
	if code != ExitOK {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	wants(t, stdout, "grid: a square lattice", "9 point(s)", "box: [0, 2] x [0, 2]",
		"all collinear: no", "index", "x", "y")
}

func TestSampleOfEveryPointSet(t *testing.T) {
	for _, kind := range []string{"grid", "circle", "spiral", "nearline", "line", "clustered"} {
		code, stdout, stderr := run("sample", "-kind", kind, "-n", "8")
		if code != ExitOK {
			t.Fatalf("%s: code = %d, stderr = %q", kind, code, stderr)
		}
		wants(t, stdout, kind+":", "8 point(s)")
	}
}

func TestPointFlagsReportBadInput(t *testing.T) {
	for _, args := range [][]string{
		{"sample", "-kind", "nope"},
		{"sample", "-n", "0"},
		{"sample", "-points", "1"},
		{"sample", "-points", "a,b"},
	} {
		code, _, stderr := run(args...)
		if code != ExitError {
			t.Fatalf("%v: code = %d, want %d", args, code, ExitError)
		}
		if stderr == "" {
			t.Fatalf("%v: stderr is empty, want an explanation", args)
		}
	}
}

func TestExplicitPoints(t *testing.T) {
	code, stdout, stderr := run("hull", "-points", "0,0;2,0;2,2;0,2;1,1")
	if code != ExitOK {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	wants(t, stdout, "5 given point(s)", "hull vertices", "4", "area", "convex", "yes")
}

func TestPointsFromAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "points.txt")
	if err := os.WriteFile(path, []byte("0,0\n2,0\n2,2\n0,2\n1,1\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	code, stdout, stderr := run("mesh", "-file", path)
	if code != ExitOK {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	wants(t, stdout, "5 point(s)", "triangles", "4", "verify", "passed")

	missing := filepath.Join(t.TempDir(), "absent.txt")
	code, _, stderr = run("mesh", "-file", missing)
	if code != ExitError {
		t.Fatalf("code = %d, want %d", code, ExitError)
	}
	wants(t, stderr, "open")

	broken := filepath.Join(t.TempDir(), "broken.txt")
	if err := os.WriteFile(broken, []byte("0,0\nnonsense\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	code, _, stderr = run("mesh", "-file", broken)
	if code != ExitError {
		t.Fatalf("code = %d, want %d", code, ExitError)
	}
	wants(t, stderr, "broken.txt")
}

func TestHullOfALattice(t *testing.T) {
	code, stdout, stderr := run("hull", "-kind", "grid", "-n", "25")
	if code != ExitOK {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	wants(t, stdout, "input points", "25", "hull vertices", "4", "perimeter", "16",
		"area", "convex", "degenerate", "no", "vertices: (0, 0)")
}

func TestHullOfCollinearPointsFindsNoArea(t *testing.T) {
	code, stdout, stderr := run("hull", "-kind", "line", "-n", "5")
	if code != ExitEmpty {
		t.Fatalf("code = %d, want %d, stderr = %q", code, ExitEmpty, stderr)
	}
	wants(t, stdout, "degenerate", "yes", "encloses no area")
}

func TestMesh(t *testing.T) {
	code, stdout, stderr := run("mesh", "-kind", "grid", "-n", "25", "-show", "3")
	if code != ExitOK {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	wants(t, stdout, "points", "25", "triangles", "32", "mesh area", "16", "hull area",
		"delaunay", "yes", "verify", "passed", "euler characteristic", "2",
		"consistent", "triangle", "area")
}

func TestMeshOfCocircularPoints(t *testing.T) {
	code, stdout, stderr := run("mesh", "-kind", "circle", "-n", "12")
	if code != ExitOK {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	wants(t, stdout, "triangles", "10", "delaunay", "yes", "verify", "passed")
}

func TestMeshReportsARepeatedPoint(t *testing.T) {
	code, _, stderr := run("mesh", "-points", "0,0;1,0;0,1;1,0")
	if code != ExitError {
		t.Fatalf("code = %d, want %d", code, ExitError)
	}
	wants(t, stderr, "repeats point")
}

func TestTopology(t *testing.T) {
	code, stdout, stderr := run("topology", "-kind", "grid", "-n", "25", "-show", "3")
	if code != ExitOK {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	wants(t, stdout, "vertices", "25", "edges", "56", "triangles", "32",
		"interior edges", "40", "largest degree", "euler characteristic", "2",
		"connected", "yes", "vertex", "degree")
}

func TestVoronoi(t *testing.T) {
	code, stdout, stderr := run("voronoi", "-kind", "grid", "-n", "25", "-show", "2")
	if code != ExitOK {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	wants(t, stdout, "sites", "25", "cells", "bounded cells", "9", "mesh area", "16",
		"coverage", "diagram:", "site", "corners")
}

func TestVoronoiOfCocircularPointsHasNoBoundedCell(t *testing.T) {
	code, stdout, stderr := run("voronoi", "-kind", "circle", "-n", "12")
	if code != ExitEmpty {
		t.Fatalf("code = %d, want %d, stderr = %q", code, ExitEmpty, stderr)
	}
	wants(t, stdout, "bounded cells", "0", "no site of this set has a bounded cell")
}

func TestQuality(t *testing.T) {
	code, stdout, stderr := run("quality", "-kind", "grid", "-n", "25", "-buckets", "6")
	if code != ExitOK {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	wants(t, stdout, "triangles", "32", "smallest angle", "45", "largest angle", "90",
		"worst aspect ratio", "areas that underflowed", "0", "slivers below",
		"worst triangle", "from", "to", "count")
}

func TestQualityFindsSlivers(t *testing.T) {
	code, stdout, stderr := run("quality", "-points", "0,0;100,0;50,0.5;25,0.2", "-sliver", "20")
	if code != ExitOK {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	wants(t, stdout, "slivers below", "slivers:")
}

func TestQualityRejectsBadFlags(t *testing.T) {
	for _, args := range [][]string{
		{"quality", "-kind", "grid", "-n", "9", "-buckets", "0"},
		{"quality", "-kind", "grid", "-n", "9", "-sliver", "0"},
		{"quality", "-kind", "grid", "-n", "9", "-sliver", "90"},
	} {
		code, _, stderr := run(args...)
		if code != ExitError {
			t.Fatalf("%v: code = %d, want %d", args, code, ExitError)
		}
		if stderr == "" {
			t.Fatalf("%v: stderr is empty, want an explanation", args)
		}
	}
}

func TestPredicates(t *testing.T) {
	code, stdout, stderr := run("predicates", "-kind", "grid", "-n", "9")
	if code != ExitOK {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	wants(t, stdout, "triples tested", "84", "collinear triples", "8",
		"filter decided", "exact fallbacks", "filtered disagreements", "0",
		"naive disagreements", "0")
}

func TestPredicatesNeedsThreePoints(t *testing.T) {
	code, _, stderr := run("predicates", "-points", "0,0;1,1")
	if code != ExitError {
		t.Fatalf("code = %d, want %d", code, ExitError)
	}
	wants(t, stderr, "at least 3 points")
}

func TestReportCoversEveryPackage(t *testing.T) {
	code, stdout, stderr := run("report")
	if code != ExitOK {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	wants(t, stdout, "exact", "geom", "hull", "delaunay", "topology", "voronoi", "quality",
		"package", "check", "value")
}
