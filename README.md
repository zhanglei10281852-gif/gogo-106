# Tessera

Tessera triangulates point sets and reads the result: a Delaunay triangulation built on
exact geometric predicates, its Voronoi dual, its connectivity, and the shape of its
worst triangle. It is written in Go with the standard library only, builds offline, and
runs as a single static binary.

This is an independent study project. It is not affiliated with, sponsored by, or
endorsed by any company, product, or organisation, and it is not a port of any
particular existing library. It exists to make one point concrete: a mesh algorithm is
only as consistent as the predicates underneath it.

Nothing here is a substitute for a production mesh generator. The example point sets are
invented, and the tolerances in the reports are chosen to be readable rather than to
certify anything.

## Quick start

```
go build -o tessera ./cmd/tessera

./tessera help
./tessera report
./tessera mesh -kind grid -n 25
./tessera predicates -kind nearline -n 8
```

Or with Docker, which builds and tests offline and produces a `scratch` image:

```
docker build -t tessera .
docker run --rm --network none tessera report
docker run --rm --network none tessera mesh -file /examples/lattice.txt
```

## Commands

| command      | what it does                                                             |
| ------------ | ------------------------------------------------------------------------ |
| `sample`     | print one of the built in point sets with its bounding box               |
| `hull`       | convex hull by monotone chain, with perimeter and area                   |
| `mesh`       | triangulate and check: area against the hull, empty circumcircles, Euler |
| `voronoi`    | the dual diagram, its bounded cells, and their area                      |
| `quality`    | angles, aspect ratios, an angle histogram and the slivers                |
| `topology`   | edges, boundary, degrees, connectivity and the counting identity         |
| `predicates` | the filtered, exact and plain answers to the orientation question        |
| `report`     | one check per package, printed side by side                              |
| `version`    | print the version                                                        |
| `help`       | print the command surface                                                |

Every geometry command takes its points in one of three ways: a built in set with
`-kind` and `-n`, an explicit list with `-points "0,0;1,0;0,1"`, or a file with `-file`.

## Exit codes

| code | meaning                                     |
| ---- | ------------------------------------------- |
| `0`  | the command ran and produced an answer      |
| `1`  | the command could not run                   |
| `2`  | the command ran and there was nothing there |

The third code says something. `hull -kind line` reports that the hull of collinear
points encloses no area, and `voronoi -kind circle` reports that no site of a cocircular
set has a bounded cell. Both are answers.

## The point sets

| kind        | what it is                                     | why it is here                                   |
| ----------- | ---------------------------------------------- | ------------------------------------------------ |
| `grid`      | a square lattice                               | every count is easy to check by hand             |
| `circle`    | points spaced evenly on a circle               | every circumcircle test comes out exactly zero   |
| `spiral`    | an arithmetic spiral                           | triangles of every shape and size                |
| `nearline`  | points a hair off a line, alternating by 1e-13 | plain arithmetic gets the orientation wrong here |
| `line`      | points exactly on a line                       | there is no triangulation, and the tool says so  |
| `clustered` | two tight clusters far apart                   | long thin triangles between the clusters         |

## What each package is for

| package             | what lives there                                                                    |
| ------------------- | ----------------------------------------------------------------------------------- |
| `internal/exact`    | the filtered orientation and in-circle predicates, with the exact rational fallback |
| `internal/geom`     | points, boxes, polygons, circumcentres and convex clipping                          |
| `internal/hull`     | the monotone chain convex hull                                                      |
| `internal/delaunay` | incremental insertion, and the verification of the result                           |
| `internal/topology` | edges, boundary, degrees, Euler and the counting identity                           |
| `internal/voronoi`  | the dual diagram, built from circumcentres, and its clipping                        |
| `internal/quality`  | angles, aspect ratios, radius-edge ratios, histograms, slivers                      |
| `internal/report`   | aligned tables and fixed digit number formatting                                    |
| `internal/cli`      | the command surface, the point set catalogue and the exit codes                     |

## The ideas worth knowing before reading the code

**Why the predicates are filtered rather than merely careful.** Which side of a line a
point falls on is a question about the sign of a determinant. Evaluated in floating
point that sign can be wrong, and worse, it can be wrong inconsistently: the same three
points may be reported as a left turn from one call site and a right turn from another,
and an algorithm that believes both builds a mesh that contradicts itself. So the
determinant is computed together with a bound on its own error. If it is larger than the
bound its sign is certain; if it is not, the determinant is computed again over the
rationals, where nothing rounds. `predicates -kind nearline -n 8` shows the difference:
the exact path is taken 8 times out of 56, plain arithmetic gets 3 of the 56 wrong, and
the filtered predicate gets none wrong.

**Why there is no enclosing triangle.** The usual presentation of incremental insertion
starts from a triangle large enough to hold every point and deletes it at the end. That
fails here: nearly collinear input makes slivers whose circumcircles are millions of
times larger than the point set, those circles swallow the enclosing vertices, and
nothing survives to the end. Instead the triangulation starts from three points of the
set and grows, and a point outside the current boundary is joined to the edges it can
see. Inserting inside and inserting outside are then the same statement: the region to
retriangulate is bounded by the symmetric difference of the cavity edges and the visible
boundary edges.

**Why a point exactly on an edge is a special case.** If the new point lies on an edge of
the region being retriangulated, joining it to that edge gives a triangle with no area,
which then has no orientation and no circumcircle. Such an edge is skipped: the point
splits it instead.

**What the checks are for.** A triangulation can be wrong in ways that are invisible in a
picture, so `mesh` re-derives the answer from several directions. The total area of the
triangles must equal the area of the convex hull. No point may lie inside any
circumcircle. Every edge must belong to one or two triangles, twice the interior edges
plus the boundary edges must equal three times the triangle count, and the Euler
characteristic must come out at two. Those checks are what caught the enclosing triangle
problem above.

**Where the exactness stops.** The combinatorics are exact; the measurements are not.
Areas, angles, circumcentres and cell polygons are floating point, and for input like
`nearline` they lose all their significance: `quality -kind nearline -n 8` reports areas
that underflowed to zero and angles of 0 and 180 degrees for triangles that the exact
predicate certifies are not degenerate. The count of underflowed areas is printed for
exactly this reason, and `voronoi -kind nearline` refuses rather than inventing
circumcentres.

## Tests

```
go vet ./...
go test ./...
```

Every package has its own tests: the predicates against their exact reference, the hull
against sets with interior, repeated and collinear points, the triangulation against
known triangle counts with full verification, the topology against the counting
identity, the diagram against the unit cells of a lattice, and the exit code of every
command.

## Repository layout

```
cmd/tessera/        the entry point
internal/           the packages listed above
examples/           invented point sets, one point per line
Dockerfile          offline multi-stage build, scratch final image
RELEASE_REPORT.md   what was verified for this release
```

No license is granted or implied by this repository.
