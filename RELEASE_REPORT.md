# Tessera release report

This report records what was checked for this release. Every result below was produced by
running the command shown, on `linux/amd64` for the container checks and on the
development host for the Go toolchain checks.

## Size

Effective lines are counted after removing blank lines and lines that are only comments.

| part                              | effective lines |
| --------------------------------- | --------------- |
| production Go (`cmd`, `internal`) | 2257            |
| tests (`*_test.go`)               | 1832            |

Per file:

| file                            | production | tests |
| ------------------------------- | ---------- | ----- |
| `cmd/tessera/main.go`           | 8          | –     |
| `internal/cli/cli.go`           | 719        | 257   |
| `internal/delaunay/delaunay.go` | 273        | 254   |
| `internal/exact/exact.go`       | 147        | 144   |
| `internal/geom/geom.go`         | 323        | 326   |
| `internal/hull/hull.go`         | 95         | 158   |
| `internal/quality/quality.go`   | 175        | 237   |
| `internal/report/report.go`     | 137        | 111   |
| `internal/topology/topology.go` | 241        | 176   |
| `internal/voronoi/voronoi.go`   | 139        | 169   |

## Toolchain checks

| check                    | result                                                    |
| ------------------------ | --------------------------------------------------------- |
| `gofmt -l internal cmd`  | no output                                                 |
| `go vet ./...`           | no findings                                               |
| `go test ./... -count=1` | ok for all nine packages, `cmd/tessera` has no test files |

What the packages cover:

- `exact`: orientations of well separated, exactly collinear and repeated points; the sign
  flip when the first two arguments swap; agreement of the filtered path with the exact
  path and of plain arithmetic with it on clean input; the in-circle predicate at the
  centre, just inside, just outside, far outside and exactly on the circle; the fourth
  corner of a square being cocircular; the counters adding up over a mix of calls and
  resetting; and every entry point working on a nil counter.
- `geom`: point arithmetic, ordering, finiteness and rendering; parsing with three
  separators and eight rejections; the orientation and in-circle wrappers; signed and
  unsigned areas; circumcentres and circumradii including the collinear refusal; boxes
  with their extent, centre, diagonal, containment, expansion, corners and rejection;
  sorted output; deduplication; first duplicate; all-collinear over seven cases; polygon
  area, perimeter and centroid including the degenerate cases; convexity including a
  reflex vertex; angular sorting with the input left alone; and convex clipping when the
  polygon is inside, overlapping, outside, degenerate or covering the box.
- `hull`: a square with an interior point, points in the middle of an edge, repeated
  points, a collinear set, one and two point sets, two rejections, containment inside, on
  the boundary and outside, and the mapping back to input indices with its rejection.
- `delaunay`: the triangle helpers; a square with a centre; a five by five lattice against
  the count 2n-2-b; twelve cocircular points; a set with three collinear points on the
  hull; the same input twice giving the same order; three rejections; clone independence;
  corners with their rejections and their orientation; five ways for `Verify` to catch a
  broken mesh; a hand built non-Delaunay mesh; and the description and longest edge.
- `topology`: edge naming; the counts of a lattice against 25 vertices, 56 edges, 32
  triangles and 40 interior edges; edges coming back in order with one or two users each;
  `Uses` handing out a copy; degrees and neighbours of the middle and of a corner;
  incident triangles in order; connectivity of three lattices and of an empty mesh; the
  Euler characteristic over four point sets; and the description.
- `voronoi`: the diagram of a lattice with 9 bounded cells of unit area and coverage nine
  sixteenths; the middle cell; an unbounded corner cell with no area; cell area with three
  rejections; two rejections in `Build`; clipping to the bounding box keeping all nine
  cells; clipping to a small box cutting them; three empty boxes rejected; and the
  description.
- `quality`: an equilateral triangle at 60 degrees; a right triangle at 45 and 90; a
  sliver; two shapeless triangles rejected; one triangle of a mesh with its rejections;
  the summary of a lattice against every field; two rejections in the summary; histograms
  including a single bucket and two rejections; slivers at three thresholds with four
  rejections; and both descriptions.
- `cli`: usage, version, an unknown command, per command help, an unknown flag, `sample`
  for all six point sets, four rejections of the point flags, explicit points, points from
  a file with a missing file and a broken file, the hull of a lattice and of a collinear
  set, `mesh` on a lattice and on cocircular points and its rejection of a repeated point,
  `topology`, `voronoi` including the cocircular set with no bounded cell, `quality` with
  slivers and three rejections, `predicates` with its rejection, and `report`.

## Container checks

| check                                                                                   | result                                                                                                 |
| --------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `docker build -t tessera:verify .`                                                      | succeeded; the builder stage ran `go vet ./...` and `go test ./...` with `GOPROXY=off` and both passed |
| `docker run --rm --network none tessera:verify report`                                  | printed all sixteen rows, matching the host run                                                        |
| `docker run --rm --network none tessera:verify mesh -file /examples/lattice.txt`        | 25 points, 32 triangles, area 16, verify passed                                                        |
| `docker run --rm --network none tessera:verify predicates -file /examples/nearline.txt` | 56 triples, 8 exact fallbacks, 0 filtered disagreements, 3 naive disagreements                         |

The final image is `scratch` and contains only the binary and `examples/`.

## Command smoke tests

The exit code column is the observed code.

| command                                        | observed result                                                                                       | exit |
| ---------------------------------------------- | ----------------------------------------------------------------------------------------------------- | ---- |
| `version`                                      | version line plus the study tool notice                                                               | 0    |
| `help`                                         | the command table, the exit codes, the point set catalogue                                            | 0    |
| `report`                                       | sixteen rows over seven packages                                                                      | 0    |
| `sample -kind grid -n 9`                       | nine lattice points, box `[0, 2] x [0, 2]`, not collinear                                             | 0    |
| `sample -kind line -n 4`                       | four points, all collinear reported as yes                                                            | 0    |
| `hull -kind grid -n 25`                        | 4 corners, perimeter 16, area 16, input indices `0 4 24 20`                                           | 0    |
| `hull -kind spiral -n 20`                      | 9 corners, perimeter 39.3414, area 113.948                                                            | 0    |
| `hull -kind line -n 5`                         | 2 vertices, degenerate, encloses no area                                                              | 2    |
| `hull -points 0,0`                             | one vertex, degenerate                                                                                | 2    |
| `mesh -kind grid -n 25`                        | 32 triangles, area 16 against a hull area of 16, delaunay, verify and consistent both passed, euler 2 | 0    |
| `mesh -kind grid -n 100`                       | 162 triangles, area 81, 36 boundary edges, euler 2                                                    | 0    |
| `mesh -kind circle -n 12`                      | 10 triangles, 12 boundary edges, area gap 4.441e-16                                                   | 0    |
| `mesh -kind nearline -n 8`                     | 8 triangles over an area of 5.99798e-13, delaunay, verify passed                                      | 0    |
| `mesh -kind nearline -n 24`                    | 28 triangles, area gap 1.832e-15, euler 2                                                             | 0    |
| `mesh -kind spiral -n 40`                      | 69 triangles, area 508.775, area gap 0                                                                | 0    |
| `mesh -kind clustered -n 24`                   | 37 triangles, area 98.3965, area gap 4.263e-14                                                        | 0    |
| `mesh -file examples/pocket.txt`               | 6 triangles, 6 boundary edges over 5 hull corners, area 14                                            | 0    |
| `mesh -kind line -n 5`                         | the points lie on one line, so there is no triangulation                                              | 1    |
| `mesh -points 0,0;1,1`                         | a triangulation needs at least three points                                                           | 1    |
| `topology -kind grid -n 25`                    | 56 edges, 40 interior, 16 boundary, largest degree 6, euler 2, connected, consistent                  | 0    |
| `topology -kind spiral -n 20`                  | 48 edges, 9 boundary, boundary walk of 9 vertices enclosing 113.948                                   | 0    |
| `topology -kind nearline -n 8`                 | 15 edges, 6 boundary, euler 2, consistent                                                             | 0    |
| `voronoi -kind grid -n 25`                     | 25 cells, 9 bounded, bounded area 9, coverage 0.5625                                                  | 0    |
| `voronoi -kind grid -n 25 -clip`               | 9 cells after clipping, area 9                                                                        | 0    |
| `voronoi -kind spiral -n 20 -clip -margin 1.2` | 11 bounded cells, area 74.3568, coverage 0.65255                                                      | 0    |
| `voronoi -kind circle -n 12`                   | no site of a cocircular set has a bounded cell                                                        | 2    |
| `voronoi -kind nearline -n 8`                  | a triangle has no circumcentre, so the diagram is refused                                             | 1    |
| `quality -kind grid -n 25`                     | every angle 45 or 90, worst aspect 1.41421, no slivers, no underflow                                  | 0    |
| `quality -kind spiral -n 20`                   | smallest angle 15.663, worst aspect 3.69743                                                           | 0    |
| `quality -kind clustered -n 24 -sliver 10`     | smallest angle 0.0102365, worst aspect 1278.23, 13 slivers                                            | 0    |
| `quality -kind nearline -n 8`                  | angles of 0 and 180, one area underflowed, all 8 triangles are slivers                                | 0    |
| `predicates -kind grid -n 9`                   | 84 triples, 8 collinear, 0 disagreements of either kind                                               | 0    |
| `predicates -kind circle -n 12`                | 220 triples, no collinear triple, no exact fallback needed                                            | 0    |
| `predicates -kind nearline -n 8`               | 56 triples, 8 exact fallbacks, 0 filtered disagreements, 3 naive disagreements                        | 0    |
| `mesh -h`                                      | the flag list on standard output                                                                      | 0    |
| `mesh -nope`                                   | `flag provided but not defined: -nope` and the flag list                                              | 1    |
| `mesh -file missing.txt`                       | the open failure named the file                                                                       | 1    |
| `sample -kind nope`                            | the unknown point set and the catalogue                                                               | 1    |
| `mush`                                         | `unknown command "mush"` and the usage screen                                                         | 1    |

## Known limitations

- The combinatorics are exact and the measurements are not. Areas, angles, circumcentres
  and cell polygons are computed in floating point, and for input that is degenerate to
  floating point they carry no information: `quality -kind nearline -n 8` reports angles of
  0 and 180 degrees and one area that underflowed to zero, for triangles the exact
  predicate certifies are not degenerate. The count of underflowed areas is printed so that
  a reader knows when to stop trusting the areas.
- `voronoi` refuses a mesh in which any triangle has no computable circumcentre, which is
  the same class of input: `voronoi -kind nearline -n 8` exits 1 rather than inventing a
  corner.
- The number of boundary edges of a triangulation equals the number of hull corners only
  when no input point lies inside a hull edge. `mesh -file examples/pocket.txt` shows 6
  boundary edges over 5 hull corners, and both numbers are right. The invariant that does
  hold is that the boundary polygon and the hull enclose the same area.
- Insertion is quadratic in the number of points: locating the cavity scans every triangle,
  and so does the boundary computation. The largest set in this report is 100 points.
- The Delaunay check is quadratic as well, since it tests every point against every
  triangle. It is a verification tool rather than part of the construction.
- A cocircular set has many valid Delaunay triangulations. The one produced here depends on
  the insertion order, which is the input order; `mesh` on the same input always gives the
  same answer, but a different order may give a different, equally valid mesh.
- Unbounded Voronoi cells carry the corners they have and are not closed. `Clip` drops them
  rather than closing them against the box, because the corner it would have to invent is
  not part of what the construction knows.
- Everything is two dimensional and everything is float64.

## Provenance

Tessera was written from scratch for this repository. There is no upstream project and no
upstream fix commit behind any part of it, so any process that requires a real upstream
repair as evidence does not apply here and is left for human review.
