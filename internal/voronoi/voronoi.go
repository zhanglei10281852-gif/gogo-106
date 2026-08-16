// Package voronoi builds the dual of a Delaunay triangulation.
//
// Every triangle of the triangulation has a circumcentre, and the circumcentres of the
// triangles around one point, taken in order, are the corners of that point's Voronoi
// cell. That is the whole construction, and it is why the two structures are always
// computed together: the hard work is the triangulation, and the diagram is a walk
// over what it already knows.
//
// The cells of points on the convex hull are unbounded, because there is no triangle on
// the far side to supply a corner. They are reported as unbounded rather than closed
// with an invented corner, and a caller who needs finite polygons clips the diagram to
// a box.
package voronoi

import (
	"fmt"
	"strings"

	"Tessera/internal/delaunay"
	"Tessera/internal/geom"
	"Tessera/internal/topology"
)

// Cell is the region of the plane closer to one site than to any other.
type Cell struct {
	// Site is the index of the point the cell belongs to.
	Site int
	// Vertices are the corners of the cell counterclockwise. An unbounded cell carries
	// the corners it does have, which do not close.
	Vertices []geom.Point
	// Bounded records whether the cell is a finite polygon.
	Bounded bool
}

// Area returns the area of a bounded cell and zero for an unbounded one.
func (c Cell) Area() float64 {
	if !c.Bounded || len(c.Vertices) < 3 {
		return 0
	}
	area := geom.PolygonArea(c.Vertices)
	if area < 0 {
		return -area
	}
	return area
}

// Diagram is the dual of a triangulation.
type Diagram struct {
	mesh  *delaunay.Mesh
	Cells []Cell
	// Clipped records whether the diagram has been cut to a box.
	Clipped bool
}

// Build returns the Voronoi diagram of a triangulation.
func Build(mesh *delaunay.Mesh) (*Diagram, error) {
	if mesh == nil {
		return nil, fmt.Errorf("a diagram needs a mesh")
	}
	if len(mesh.Triangles) == 0 {
		return nil, fmt.Errorf("a diagram needs a mesh with triangles")
	}
	graph := topology.Build(mesh)
	centres := make([]geom.Point, len(mesh.Triangles))
	for index := range mesh.Triangles {
		corners, err := mesh.Corners(index)
		if err != nil {
			return nil, err
		}
		centre, ok := geom.Circumcenter(corners[0], corners[1], corners[2])
		if !ok {
			return nil, fmt.Errorf("triangle %d has no circumcentre, so the mesh is degenerate", index)
		}
		centres[index] = centre
	}
	out := &Diagram{mesh: mesh}
	boundary := map[int]bool{}
	for _, vertex := range graph.BoundaryVertices() {
		boundary[vertex] = true
	}
	for site := range mesh.Points {
		incident := graph.IncidentTriangles(site)
		if len(incident) == 0 {
			continue
		}
		corners := make([]geom.Point, 0, len(incident))
		for _, triangle := range incident {
			corners = append(corners, centres[triangle])
		}
		cell := Cell{
			Site:     site,
			Vertices: geom.SortAround(mesh.Points[site], corners),
			Bounded:  !boundary[site],
		}
		out.Cells = append(out.Cells, cell)
	}
	return out, nil
}

// Mesh returns the triangulation the diagram was built from.
func (d *Diagram) Mesh() *delaunay.Mesh { return d.mesh }

// Cell returns the cell of a site.
func (d *Diagram) Cell(site int) (Cell, bool) {
	for _, cell := range d.Cells {
		if cell.Site == site {
			return cell, true
		}
	}
	return Cell{}, false
}

// Bounded returns how many cells are finite polygons.
func (d *Diagram) Bounded() int {
	count := 0
	for _, cell := range d.Cells {
		if cell.Bounded {
			count++
		}
	}
	return count
}

// TotalArea returns the area of the bounded cells.
func (d *Diagram) TotalArea() float64 {
	total := 0.0
	for _, cell := range d.Cells {
		total += cell.Area()
	}
	return total
}

// CellArea returns the area of one site's cell. The site is checked against the mesh
// the diagram was built from, so an index from somewhere else is reported rather than
// silently answered.
func (d *Diagram) CellArea(site int) (float64, error) {
	if site < 0 || site >= len(d.mesh.Points) {
		return 0, fmt.Errorf("site %d is not one of the %d points of the mesh", site, len(d.mesh.Points))
	}
	cell, ok := d.Cell(site)
	if !ok {
		return 0, fmt.Errorf("site %d has no cell", site)
	}
	return cell.Area(), nil
}

// Clip returns the diagram cut to a box. Only the bounded cells survive: an unbounded
// cell has no polygon to cut, and inventing one would put area where the construction
// says nothing.
func (d *Diagram) Clip(box geom.Box) (*Diagram, error) {
	if box.Width() <= 0 || box.Height() <= 0 {
		return nil, fmt.Errorf("the clip box %s is empty", box)
	}
	out := &Diagram{mesh: d.mesh, Clipped: true}
	for _, cell := range d.Cells {
		if !cell.Bounded {
			continue
		}
		clipped := geom.ClipConvex(cell.Vertices, box)
		if len(clipped) < 3 {
			continue
		}
		out.Cells = append(out.Cells, Cell{Site: cell.Site, Vertices: clipped, Bounded: true})
	}
	return out, nil
}

// Describe renders the diagram for a report, against the mesh it came from.
func (d *Diagram) Describe() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%d cell(s) over %d site(s), %d bounded, area %.6g",
		len(d.Cells), len(d.mesh.Points), d.Bounded(), d.TotalArea())
	if d.Clipped {
		builder.WriteString(", clipped")
	}
	return builder.String()
}

// Coverage returns the ratio of the bounded cell area to the area of the triangulation,
// which says how much of the hull the finite cells account for.
func (d *Diagram) Coverage() float64 {
	area := d.mesh.Area()
	if area == 0 {
		return 0
	}
	return d.TotalArea() / area
}
