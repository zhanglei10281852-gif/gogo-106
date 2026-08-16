// Package cli is the Tessera command surface.
//
// Exit codes are part of the contract: 0 means the command ran and produced an answer,
// 1 means it could not run, and 2 means it ran and there was nothing there. The last
// one carries information in this subject: a point set whose points are all on one line
// has a hull with no area, and saying so is an answer rather than a failure.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"

	"Tessera/internal/delaunay"
	"Tessera/internal/exact"
	"Tessera/internal/geom"
	"Tessera/internal/hull"
	"Tessera/internal/quality"
	"Tessera/internal/report"
	"Tessera/internal/topology"
	"Tessera/internal/voronoi"
)

// Version identifies the build.
const Version = "tessera 1.0.0"

// Exit codes.
const (
	ExitOK    = 0
	ExitError = 1
	ExitEmpty = 2
)

// command is one subcommand.
type command struct {
	name    string
	summary string
	run     func(args []string, stdout, stderr io.Writer) (int, error)
}

// commands returns the subcommand table.
func commands() map[string]command {
	out := map[string]command{}
	for _, item := range []command{
		{"sample", "print one of the built in point sets", runSample},
		{"hull", "compute the convex hull of a point set", runHull},
		{"mesh", "triangulate a point set and check the result", runMesh},
		{"voronoi", "build the dual diagram of a triangulation", runVoronoi},
		{"quality", "measure the shape of the triangles of a mesh", runQuality},
		{"topology", "read the connectivity of a mesh", runTopology},
		{"predicates", "compare the filtered, exact and naive predicates", runPredicates},
		{"report", "run one check per package and print the numbers", runReport},
	} {
		out[item.name] = item
	}
	return out
}

// Run dispatches one invocation.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return ExitError
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, Version)
		fmt.Fprintln(stdout, "study tool only: a teaching implementation of robust mesh generation")
		return ExitOK
	case "help", "--help", "-h":
		usage(stdout)
		return ExitOK
	}
	chosen, known := commands()[args[0]]
	if !known {
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		usage(stderr)
		return ExitError
	}
	code, err := chosen.run(args[1:], stdout, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ExitError
	}
	return code
}

// usage prints the command surface.
func usage(writer io.Writer) {
	fmt.Fprintln(writer, Version)
	fmt.Fprintln(writer, "study tool only: a teaching implementation of robust mesh generation")
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "usage: tessera <command> [flags]")
	fmt.Fprintln(writer, "")
	table := report.Table{Gap: 2}
	names := make([]string, 0, len(commands()))
	for name := range commands() {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		table.AddRow("  "+name, commands()[name].summary)
	}
	fmt.Fprint(writer, table.Render())
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "exit codes: 0 produced an answer, 1 could not run, 2 ran and found nothing")
	fmt.Fprintln(writer, "point sets: "+strings.Join(generatorNames(), ", "))
}

// parse reads the flags of one subcommand. A request for help is a successful run
// rather than a failure, and a bad flag reports itself once instead of once per layer.
func parse(set *flag.FlagSet, args []string, stdout, stderr io.Writer) (int, bool) {
	set.SetOutput(io.Discard)
	err := set.Parse(args)
	switch {
	case err == nil:
		return ExitOK, true
	case errors.Is(err, flag.ErrHelp):
		fmt.Fprintf(stdout, "%s - %s\n\n", set.Name(), commands()[set.Name()].summary)
		set.SetOutput(stdout)
		set.PrintDefaults()
		return ExitOK, false
	default:
		fmt.Fprintln(stderr, err)
		set.SetOutput(stderr)
		set.PrintDefaults()
		return ExitError, false
	}
}

// generator builds one of the built in point sets.
type generator struct {
	summary string
	build   func(count int) ([]geom.Point, error)
}

// generators returns the point set catalogue. Every set is a fixed function of its
// size, so a result can be reproduced by anyone who runs the same command.
func generators() map[string]generator {
	return map[string]generator{
		"grid": {"a square lattice, row by row", func(count int) ([]geom.Point, error) {
			side := int(math.Ceil(math.Sqrt(float64(count))))
			out := make([]geom.Point, 0, count)
			for row := 0; row < side && len(out) < count; row++ {
				for column := 0; column < side && len(out) < count; column++ {
					out = append(out, geom.Point{X: float64(column), Y: float64(row)})
				}
			}
			return out, nil
		}},
		"circle": {"points spaced evenly on a circle, which are all cocircular",
			func(count int) ([]geom.Point, error) {
				out := make([]geom.Point, 0, count)
				for index := 0; index < count; index++ {
					angle := 2 * math.Pi * float64(index) / float64(count)
					out = append(out, geom.Point{X: math.Cos(angle), Y: math.Sin(angle)})
				}
				return out, nil
			}},
		"spiral": {"an arithmetic spiral", func(count int) ([]geom.Point, error) {
			out := make([]geom.Point, 0, count)
			for index := 0; index < count; index++ {
				angle := 0.7 * float64(index)
				radius := 1 + 0.35*float64(index)
				out = append(out, geom.Point{X: radius * math.Cos(angle), Y: radius * math.Sin(angle)})
			}
			return out, nil
		}},
		"nearline": {"points a hair off a straight line, alternating by 1e-13",
			func(count int) ([]geom.Point, error) {
				out := make([]geom.Point, 0, count)
				for index := 0; index < count; index++ {
					x := float64(index)
					y := 0.5 * x
					if index%2 == 1 {
						y += 1e-13
					}
					out = append(out, geom.Point{X: x, Y: y})
				}
				return out, nil
			}},
		"line": {"points exactly on a straight line, which have no triangulation",
			func(count int) ([]geom.Point, error) {
				out := make([]geom.Point, 0, count)
				for index := 0; index < count; index++ {
					out = append(out, geom.Point{X: float64(index), Y: 2*float64(index) + 1})
				}
				return out, nil
			}},
		"clustered": {"two tight clusters far apart", func(count int) ([]geom.Point, error) {
			out := make([]geom.Point, 0, count)
			for index := 0; index < count; index++ {
				side := float64(index % 2)
				step := float64(index / 2)
				out = append(out, geom.Point{
					X: side*100 + 0.5*math.Cos(step),
					Y: 0.5 * math.Sin(1.7*step),
				})
			}
			return out, nil
		}},
	}
}

// generatorNames returns the catalogue names in order.
func generatorNames() []string {
	out := make([]string, 0, len(generators()))
	for name := range generators() {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// pointFlags declares the flags every geometry command shares.
type pointFlags struct {
	kind     *string
	count    *int
	explicit *string
	file     *string
}

// declarePoints adds the point set flags to a set.
func declarePoints(set *flag.FlagSet) pointFlags {
	return pointFlags{
		kind:     set.String("kind", "grid", "built in point set to use"),
		count:    set.Int("n", 16, "how many points to generate"),
		explicit: set.String("points", "", "explicit point list, x,y pairs separated by semicolons"),
		file:     set.String("file", "", "read the point list from a file instead"),
	}
}

// resolve returns the points a command should work on, together with a label naming
// where they came from.
func (p pointFlags) resolve() ([]geom.Point, string, error) {
	if *p.file != "" {
		content, err := os.ReadFile(*p.file)
		if err != nil {
			// The error from os.ReadFile already names the file and the operation.
			return nil, "", err
		}
		points, err := geom.Parse(string(content))
		if err != nil {
			return nil, "", fmt.Errorf("%s: %w", *p.file, err)
		}
		return points, fmt.Sprintf("%s: %d point(s)", *p.file, len(points)), nil
	}
	if *p.explicit != "" {
		points, err := geom.Parse(*p.explicit)
		if err != nil {
			return nil, "", err
		}
		return points, fmt.Sprintf("%d given point(s)", len(points)), nil
	}
	chosen, known := generators()[*p.kind]
	if !known {
		return nil, "", fmt.Errorf("unknown point set %q; the catalogue is %s",
			*p.kind, strings.Join(generatorNames(), ", "))
	}
	if *p.count < 1 {
		return nil, "", fmt.Errorf("a point set needs a size of 1 or more, got %d", *p.count)
	}
	points, err := chosen.build(*p.count)
	if err != nil {
		return nil, "", err
	}
	return points, fmt.Sprintf("%s: %s (%d point(s))", *p.kind, chosen.summary, len(points)), nil
}

// runSample prints a point set.
func runSample(args []string, stdout, stderr io.Writer) (int, error) {
	set := flag.NewFlagSet("sample", flag.ContinueOnError)
	flags := declarePoints(set)
	if code, ok := parse(set, args, stdout, stderr); !ok {
		return code, nil
	}
	points, label, err := flags.resolve()
	if err != nil {
		return ExitError, err
	}
	box, err := geom.BoxOf(points)
	if err != nil {
		return ExitError, err
	}
	fmt.Fprintf(stdout, "%s\n", label)
	fmt.Fprintf(stdout, "box: %s\n", box)
	fmt.Fprintf(stdout, "all collinear: %s\n\n", report.Bool(geom.AllCollinear(points)))
	table := report.Table{
		Header:     []string{"index", "x", "y"},
		Alignments: []report.Alignment{report.AlignRight, report.AlignRight, report.AlignRight},
	}
	for index, point := range points {
		table.AddRow(report.Int(index), report.Float(point.X), report.Float(point.Y))
	}
	fmt.Fprint(stdout, table.Render())
	return ExitOK, nil
}

// runHull computes a convex hull.
func runHull(args []string, stdout, stderr io.Writer) (int, error) {
	set := flag.NewFlagSet("hull", flag.ContinueOnError)
	flags := declarePoints(set)
	if code, ok := parse(set, args, stdout, stderr); !ok {
		return code, nil
	}
	points, label, err := flags.resolve()
	if err != nil {
		return ExitError, err
	}
	vertices, err := hull.Hull(points)
	if err != nil {
		return ExitError, err
	}
	indices, err := hull.IndexIn(points, vertices)
	if err != nil {
		return ExitError, err
	}
	fmt.Fprintf(stdout, "%s\n", label)
	table := report.Table{
		Header:     []string{"measure", "value"},
		Alignments: []report.Alignment{report.AlignLeft, report.AlignRight},
	}
	table.AddRow("input points", report.Int(len(points)))
	table.AddRow("hull vertices", report.Int(len(vertices)))
	table.AddRow("input indices", report.Ints(indices))
	table.AddRow("perimeter", report.Float(hull.Perimeter(vertices)))
	table.AddRow("area", report.Float(hull.Area(vertices)))
	table.AddRow("convex", report.Bool(geom.IsConvex(vertices)))
	table.AddRow("degenerate", report.Bool(hull.Degenerate(vertices)))
	fmt.Fprint(stdout, table.Render())
	fmt.Fprintf(stdout, "\nvertices: %s\n", renderPoints(vertices))
	if hull.Degenerate(vertices) {
		fmt.Fprintf(stdout, "\nthe hull of these points encloses no area\n")
		return ExitEmpty, nil
	}
	return ExitOK, nil
}

// renderPoints renders a short point list.
func renderPoints(points []geom.Point) string {
	parts := make([]string, 0, len(points))
	for _, point := range points {
		parts = append(parts, point.String())
	}
	return strings.Join(parts, " ")
}

// runMesh triangulates a point set and checks the result.
func runMesh(args []string, stdout, stderr io.Writer) (int, error) {
	set := flag.NewFlagSet("mesh", flag.ContinueOnError)
	flags := declarePoints(set)
	show := set.Int("show", 0, "print this many triangles")
	if code, ok := parse(set, args, stdout, stderr); !ok {
		return code, nil
	}
	points, label, err := flags.resolve()
	if err != nil {
		return ExitError, err
	}
	mesh, err := delaunay.Triangulate(points)
	if err != nil {
		return ExitError, err
	}
	vertices, err := hull.Hull(points)
	if err != nil {
		return ExitError, err
	}
	graph := topology.Build(mesh)
	fmt.Fprintf(stdout, "%s\n", label)
	table := report.Table{
		Header:     []string{"measure", "value"},
		Alignments: []report.Alignment{report.AlignLeft, report.AlignRight},
	}
	table.AddRow("points", report.Int(len(mesh.Points)))
	table.AddRow("triangles", report.Int(len(mesh.Triangles)))
	table.AddRow("used vertices", report.Int(mesh.UsedVertices()))
	table.AddRow("mesh area", report.Float(mesh.Area()))
	table.AddRow("hull area", report.Float(hull.Area(vertices)))
	table.AddRow("area gap", report.Sci(math.Abs(mesh.Area()-hull.Area(vertices))))
	_, delaunayOK := mesh.IsDelaunay()
	table.AddRow("delaunay", report.Bool(delaunayOK))
	if err := mesh.Verify(); err != nil {
		table.AddRow("verify", err.Error())
	} else {
		table.AddRow("verify", "passed")
	}
	table.AddRow("boundary edges", report.Int(len(graph.BoundaryEdges())))
	table.AddRow("hull corners", report.Int(len(vertices)))
	// The boundary of the triangulation is the hull, but it has one edge per point that
	// lies on the hull rather than one per corner, so the two counts agree only when no
	// point sits inside a hull edge.
	if polygon, err := graph.BoundaryPolygon(); err == nil {
		table.AddRow("boundary polygon area", report.Float(math.Abs(geom.PolygonArea(polygon))))
	} else {
		table.AddRow("boundary polygon area", err.Error())
	}
	table.AddRow("interior edges", report.Int(len(graph.InteriorEdges())))
	table.AddRow("euler characteristic", report.Int(graph.Euler()))
	if err := graph.Consistent(); err != nil {
		table.AddRow("consistent", err.Error())
	} else {
		table.AddRow("consistent", "passed")
	}
	fmt.Fprint(stdout, table.Render())
	if *show > 0 {
		limit := *show
		if limit > len(mesh.Triangles) {
			limit = len(mesh.Triangles)
		}
		block := report.Table{
			Header: []string{"triangle", "a", "b", "c", "area"},
			Alignments: []report.Alignment{
				report.AlignRight, report.AlignRight, report.AlignRight,
				report.AlignRight, report.AlignRight,
			},
		}
		for index := 0; index < limit; index++ {
			triangle := mesh.Triangles[index]
			corners, err := mesh.Corners(index)
			if err != nil {
				return ExitError, err
			}
			block.AddRow(report.Int(index), report.Int(triangle.A), report.Int(triangle.B),
				report.Int(triangle.C), report.Float(geom.TriangleArea(corners[0], corners[1], corners[2])))
		}
		fmt.Fprint(stdout, "\n"+block.Render())
	}
	return ExitOK, nil
}

// runVoronoi builds the dual diagram.
func runVoronoi(args []string, stdout, stderr io.Writer) (int, error) {
	set := flag.NewFlagSet("voronoi", flag.ContinueOnError)
	flags := declarePoints(set)
	clip := set.Bool("clip", false, "cut the diagram to the bounding box of the points")
	margin := set.Float64("margin", 1.5, "how far to expand the box before clipping")
	show := set.Int("show", 0, "print the area of this many cells")
	if code, ok := parse(set, args, stdout, stderr); !ok {
		return code, nil
	}
	points, label, err := flags.resolve()
	if err != nil {
		return ExitError, err
	}
	mesh, err := delaunay.Triangulate(points)
	if err != nil {
		return ExitError, err
	}
	diagram, err := voronoi.Build(mesh)
	if err != nil {
		return ExitError, err
	}
	fmt.Fprintf(stdout, "%s\n", label)
	if *clip {
		box, err := geom.BoxOf(points)
		if err != nil {
			return ExitError, err
		}
		diagram, err = diagram.Clip(box.Expand(*margin))
		if err != nil {
			return ExitError, err
		}
	}
	table := report.Table{
		Header:     []string{"measure", "value"},
		Alignments: []report.Alignment{report.AlignLeft, report.AlignRight},
	}
	table.AddRow("sites", report.Int(len(points)))
	table.AddRow("cells", report.Int(len(diagram.Cells)))
	table.AddRow("bounded cells", report.Int(diagram.Bounded()))
	table.AddRow("clipped", report.Bool(*clip))
	table.AddRow("bounded area", report.Float(diagram.TotalArea()))
	table.AddRow("mesh area", report.Float(mesh.Area()))
	table.AddRow("coverage", report.Float(diagram.Coverage()))
	fmt.Fprint(stdout, table.Render())
	fmt.Fprintf(stdout, "\ndiagram: %s\n", diagram.Describe())
	if *show > 0 {
		block := report.Table{
			Header: []string{"site", "corners", "bounded", "area"},
			Alignments: []report.Alignment{
				report.AlignRight, report.AlignRight, report.AlignRight, report.AlignRight,
			},
		}
		limit := *show
		if limit > len(diagram.Cells) {
			limit = len(diagram.Cells)
		}
		for _, cell := range diagram.Cells[:limit] {
			area, err := diagram.CellArea(cell.Site)
			if err != nil {
				return ExitError, err
			}
			block.AddRow(report.Int(cell.Site), report.Int(len(cell.Vertices)),
				report.Bool(cell.Bounded), report.Float(area))
		}
		fmt.Fprint(stdout, "\n"+block.Render())
	}
	if diagram.Bounded() == 0 {
		fmt.Fprintf(stdout, "\nno site of this set has a bounded cell\n")
		return ExitEmpty, nil
	}
	return ExitOK, nil
}

// runQuality measures the shape of the triangles.
func runQuality(args []string, stdout, stderr io.Writer) (int, error) {
	set := flag.NewFlagSet("quality", flag.ContinueOnError)
	flags := declarePoints(set)
	buckets := set.Int("buckets", 12, "how many buckets the angle histogram has")
	sliver := set.Float64("sliver", 15, "smallest angle in degrees below which a triangle is a sliver")
	if code, ok := parse(set, args, stdout, stderr); !ok {
		return code, nil
	}
	points, label, err := flags.resolve()
	if err != nil {
		return ExitError, err
	}
	mesh, err := delaunay.Triangulate(points)
	if err != nil {
		return ExitError, err
	}
	summary, err := quality.Summarise(mesh)
	if err != nil {
		return ExitError, err
	}
	counts, err := quality.Histogram(mesh, *buckets)
	if err != nil {
		return ExitError, err
	}
	slivers, err := quality.Slivers(mesh, *sliver)
	if err != nil {
		return ExitError, err
	}
	fmt.Fprintf(stdout, "%s\n", label)
	table := report.Table{
		Header:     []string{"measure", "value"},
		Alignments: []report.Alignment{report.AlignLeft, report.AlignRight},
	}
	table.AddRow("triangles", report.Int(summary.Triangles))
	table.AddRow("smallest angle", report.Float(summary.MinAngle))
	table.AddRow("largest angle", report.Float(summary.MaxAngle))
	table.AddRow("mean smallest angle", report.Float(summary.MeanMinAngle))
	table.AddRow("smallest area", report.Float(summary.MinArea))
	table.AddRow("largest area", report.Float(summary.MaxArea))
	table.AddRow("worst aspect ratio", report.Float(summary.WorstAspect))
	table.AddRow("worst radius-edge ratio", report.Float(summary.WorstRadiusEdge))
	table.AddRow("worst triangle", report.Int(summary.Worst))
	table.AddRow("areas that underflowed", report.Int(summary.Underflowed))
	table.AddRow("slivers below "+report.Float(*sliver), report.Int(len(slivers)))
	fmt.Fprint(stdout, table.Render())
	worst, err := quality.Of(mesh, summary.Worst)
	if err != nil {
		return ExitError, err
	}
	fmt.Fprintf(stdout, "\nworst triangle %d: %s\n", summary.Worst, worst.Describe())
	largest := 0
	for _, count := range counts {
		if count > largest {
			largest = count
		}
	}
	block := report.Table{
		Header: []string{"from", "to", "count", ""},
		Alignments: []report.Alignment{
			report.AlignRight, report.AlignRight, report.AlignRight, report.AlignLeft,
		},
	}
	width := 60.0 / float64(*buckets)
	for index, count := range counts {
		block.AddRow(
			report.Float(float64(index)*width),
			report.Float(float64(index+1)*width),
			report.Int(count),
			report.Bar(count, largest, 30))
	}
	fmt.Fprint(stdout, "\n"+block.Render())
	if len(slivers) == 0 {
		return ExitOK, nil
	}
	fmt.Fprintf(stdout, "\nslivers: %s\n", report.Ints(slivers))
	return ExitOK, nil
}

// runTopology reads the connectivity of a mesh.
func runTopology(args []string, stdout, stderr io.Writer) (int, error) {
	set := flag.NewFlagSet("topology", flag.ContinueOnError)
	flags := declarePoints(set)
	show := set.Int("show", 0, "print the degree of this many vertices")
	if code, ok := parse(set, args, stdout, stderr); !ok {
		return code, nil
	}
	points, label, err := flags.resolve()
	if err != nil {
		return ExitError, err
	}
	mesh, err := delaunay.Triangulate(points)
	if err != nil {
		return ExitError, err
	}
	graph := topology.Build(mesh)
	fmt.Fprintf(stdout, "%s\n", label)
	table := report.Table{
		Header:     []string{"measure", "value"},
		Alignments: []report.Alignment{report.AlignLeft, report.AlignRight},
	}
	table.AddRow("vertices", report.Int(graph.VertexCount()))
	table.AddRow("edges", report.Int(graph.EdgeCount()))
	table.AddRow("triangles", report.Int(graph.TriangleCount()))
	table.AddRow("interior edges", report.Int(len(graph.InteriorEdges())))
	table.AddRow("boundary edges", report.Int(len(graph.BoundaryEdges())))
	table.AddRow("overused edges", report.Int(len(graph.OverusedEdges())))
	table.AddRow("boundary vertices", report.Int(len(graph.BoundaryVertices())))
	table.AddRow("boundary length", report.Float(graph.BoundaryLength()))
	table.AddRow("largest degree", report.Int(graph.MaxDegree()))
	table.AddRow("euler characteristic", report.Int(graph.Euler()))
	table.AddRow("connected", report.Bool(graph.Connected()))
	if err := graph.Consistent(); err != nil {
		table.AddRow("consistent", err.Error())
	} else {
		table.AddRow("consistent", "passed")
	}
	polygon, err := graph.BoundaryPolygon()
	if err != nil {
		table.AddRow("boundary walk", err.Error())
	} else {
		table.AddRow("boundary walk", report.Int(len(polygon))+" vertex(es)")
		table.AddRow("boundary polygon area", report.Float(math.Abs(geom.PolygonArea(polygon))))
	}
	fmt.Fprint(stdout, table.Render())
	if *show > 0 {
		limit := *show
		if limit > len(mesh.Points) {
			limit = len(mesh.Points)
		}
		block := report.Table{
			Header: []string{"vertex", "degree", "triangles", "boundary"},
			Alignments: []report.Alignment{
				report.AlignRight, report.AlignRight, report.AlignRight, report.AlignRight,
			},
		}
		for vertex := 0; vertex < limit; vertex++ {
			block.AddRow(report.Int(vertex), report.Int(graph.Degree(vertex)),
				report.Int(len(graph.IncidentTriangles(vertex))),
				report.Bool(graph.IsBoundaryVertex(vertex)))
		}
		fmt.Fprint(stdout, "\n"+block.Render())
	}
	return ExitOK, nil
}

// runPredicates compares the three ways of deciding an orientation.
func runPredicates(args []string, stdout, stderr io.Writer) (int, error) {
	set := flag.NewFlagSet("predicates", flag.ContinueOnError)
	flags := declarePoints(set)
	if code, ok := parse(set, args, stdout, stderr); !ok {
		return code, nil
	}
	points, label, err := flags.resolve()
	if err != nil {
		return ExitError, err
	}
	if len(points) < 3 {
		return ExitError, fmt.Errorf("a comparison needs at least 3 points, got %d", len(points))
	}
	var predicates exact.Predicates
	naiveDisagreements := 0
	filteredDisagreements := 0
	collinear := 0
	triples := 0
	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			for k := j + 1; k < len(points); k++ {
				a, b, c := points[i], points[j], points[k]
				truth := exact.ExactOrient2D(a.X, a.Y, b.X, b.Y, c.X, c.Y)
				filtered := predicates.Orient2D(a.X, a.Y, b.X, b.Y, c.X, c.Y)
				naive := exact.NaiveOrient2D(a.X, a.Y, b.X, b.Y, c.X, c.Y)
				if filtered != truth {
					filteredDisagreements++
				}
				if naive != truth {
					naiveDisagreements++
				}
				if truth == exact.Collinear {
					collinear++
				}
				triples++
			}
		}
	}
	stats := predicates.Stats()
	fmt.Fprintf(stdout, "%s\n", label)
	table := report.Table{
		Header:     []string{"measure", "value"},
		Alignments: []report.Alignment{report.AlignLeft, report.AlignRight},
	}
	table.AddRow("triples tested", report.Int(triples))
	table.AddRow("collinear triples", report.Int(collinear))
	table.AddRow("filter decided", report.Int(stats.Filtered))
	table.AddRow("exact fallbacks", report.Int(stats.Exact))
	table.AddRow("filtered disagreements", report.Int(filteredDisagreements))
	table.AddRow("naive disagreements", report.Int(naiveDisagreements))
	fmt.Fprint(stdout, table.Render())
	if filteredDisagreements > 0 {
		fmt.Fprintf(stdout, "\nthe filtered predicate disagrees with the exact one on %d triple(s)\n",
			filteredDisagreements)
	}
	return ExitOK, nil
}

// runReport runs one check per package and prints the numbers side by side.
func runReport(args []string, stdout, stderr io.Writer) (int, error) {
	set := flag.NewFlagSet("report", flag.ContinueOnError)
	if code, ok := parse(set, args, stdout, stderr); !ok {
		return code, nil
	}
	table := report.Table{
		Header:     []string{"package", "check", "value"},
		Alignments: []report.Alignment{report.AlignLeft, report.AlignLeft, report.AlignRight},
	}

	// Three points a hair off a line, where plain floating point arithmetic gets the
	// orientation wrong and the filtered predicate does not.
	ax, ay := 0.5, 0.5
	bx, by := 12.0, 12.0
	cx, cy := 24.0, 24.000000000000004
	table.AddRow("exact", "orientation of a near degenerate triple",
		report.Int(exact.Orient2D(ax, ay, bx, by, cx, cy)))
	table.AddRow("exact", "the same triple by plain arithmetic",
		report.Int(exact.NaiveOrient2D(ax, ay, bx, by, cx, cy)))

	square := []geom.Point{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}
	table.AddRow("geom", "area of the unit square", report.Float(geom.PolygonArea(square)))
	centre, ok := geom.Circumcenter(square[0], square[1], square[2])
	table.AddRow("geom", "circumcentre of three of its corners", centre.String())
	table.AddRow("geom", "that circumcentre exists", report.Bool(ok))

	grid, err := generators()["grid"].build(25)
	if err != nil {
		return ExitError, err
	}
	vertices, err := hull.Hull(grid)
	if err != nil {
		return ExitError, err
	}
	table.AddRow("hull", "hull vertices of a 5 by 5 lattice", report.Int(len(vertices)))
	table.AddRow("hull", "hull area of the same lattice", report.Float(hull.Area(vertices)))

	mesh, err := delaunay.Triangulate(grid)
	if err != nil {
		return ExitError, err
	}
	table.AddRow("delaunay", "triangles of the lattice", report.Int(len(mesh.Triangles)))
	if err := mesh.Verify(); err != nil {
		table.AddRow("delaunay", "verify", err.Error())
	} else {
		table.AddRow("delaunay", "verify", "passed")
	}

	graph := topology.Build(mesh)
	table.AddRow("topology", "boundary edges of the lattice", report.Int(len(graph.BoundaryEdges())))
	if err := graph.Consistent(); err != nil {
		table.AddRow("topology", "consistent", err.Error())
	} else {
		table.AddRow("topology", "consistent", "passed")
	}

	diagram, err := voronoi.Build(mesh)
	if err != nil {
		return ExitError, err
	}
	table.AddRow("voronoi", "bounded cells of the lattice", report.Int(diagram.Bounded()))
	table.AddRow("voronoi", "area of the bounded cells", report.Float(diagram.TotalArea()))

	summary, err := quality.Summarise(mesh)
	if err != nil {
		return ExitError, err
	}
	table.AddRow("quality", "smallest angle in degrees", report.Float(summary.MinAngle))
	table.AddRow("quality", "worst aspect ratio", report.Float(summary.WorstAspect))

	fmt.Fprint(stdout, table.Render())
	return ExitOK, nil
}
