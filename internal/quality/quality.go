// Package quality measures how well shaped the triangles of a mesh are.
//
// A triangulation can be correct and still useless. What makes it usable is the shape
// of its worst triangle: a sliver with a one degree angle ruins the accuracy of
// anything interpolated across it, however Delaunay the mesh is. The numbers here are
// the ones that catch slivers, and the smallest angle is the one to read first.
package quality

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"Tessera/internal/delaunay"
	"Tessera/internal/geom"
)

// Metrics describes the shape of one triangle.
type Metrics struct {
	// Area is the area of the triangle.
	Area float64
	// MinAngle and MaxAngle are in degrees.
	MinAngle float64
	MaxAngle float64
	// ShortestEdge and LongestEdge are lengths.
	ShortestEdge float64
	LongestEdge  float64
	// AspectRatio is the longest edge over the shortest.
	AspectRatio float64
	// RadiusEdgeRatio is the circumradius over the shortest edge, the measure a mesh
	// generator actually bounds. It is infinite when the circumradius is too large to
	// compute in floating point.
	RadiusEdgeRatio float64
	// AreaUnderflowed records that the area rounded to zero even though the exact
	// predicate says the three points are not collinear. It happens for input that is
	// degenerate to floating point but not degenerate in fact, and it means the area
	// here carries no information while the angles still do.
	AreaUnderflowed bool
}

// Of returns the metrics of one triangle of a mesh.
func Of(mesh *delaunay.Mesh, index int) (Metrics, error) {
	corners, err := mesh.Corners(index)
	if err != nil {
		return Metrics{}, err
	}
	return OfPoints(corners[0], corners[1], corners[2])
}

// OfPoints returns the metrics of a triangle given by its corners.
func OfPoints(a, b, c geom.Point) (Metrics, error) {
	lengths := []float64{b.Dist(c), c.Dist(a), a.Dist(b)}
	for _, length := range lengths {
		if length == 0 {
			return Metrics{}, fmt.Errorf("a triangle with a zero length edge has no shape")
		}
	}
	if geom.Collinear(a, b, c) {
		return Metrics{}, fmt.Errorf("a triangle whose points are collinear has no shape")
	}
	area := geom.TriangleArea(a, b, c)
	angles := []float64{
		angleAt(lengths[0], lengths[1], lengths[2]),
		angleAt(lengths[1], lengths[2], lengths[0]),
		angleAt(lengths[2], lengths[0], lengths[1]),
	}
	sort.Float64s(angles)
	sorted := append([]float64(nil), lengths...)
	sort.Float64s(sorted)
	// The circumradius comes from a determinant that cancels for a sliver, so it is not
	// always available. The angles come from the edge lengths alone and always are.
	ratio := math.Inf(1)
	if radius, ok := geom.Circumradius(a, b, c); ok {
		ratio = radius / sorted[0]
	}
	return Metrics{
		Area:            area,
		MinAngle:        angles[0],
		MaxAngle:        angles[2],
		ShortestEdge:    sorted[0],
		LongestEdge:     sorted[2],
		AspectRatio:     sorted[2] / sorted[0],
		RadiusEdgeRatio: ratio,
		AreaUnderflowed: area == 0,
	}, nil
}

// angleAt returns the angle in degrees opposite the first length, by the law of
// cosines. The ratio is clamped because a rounded value just outside the valid range
// would otherwise come back as a not-a-number.
func angleAt(opposite, left, right float64) float64 {
	ratio := (left*left + right*right - opposite*opposite) / (2 * left * right)
	if ratio > 1 {
		ratio = 1
	}
	if ratio < -1 {
		ratio = -1
	}
	return math.Acos(ratio) * 180 / math.Pi
}

// Describe renders the metrics of one triangle.
func (m Metrics) Describe() string {
	return fmt.Sprintf("area %.6g, angles %.4g to %.4g degrees, aspect %.4g, radius-edge %.4g",
		m.Area, m.MinAngle, m.MaxAngle, m.AspectRatio, m.RadiusEdgeRatio)
}

// Summary describes the shape of a whole mesh.
type Summary struct {
	Triangles       int
	MinAngle        float64
	MaxAngle        float64
	MeanMinAngle    float64
	MinArea         float64
	MaxArea         float64
	TotalArea       float64
	WorstAspect     float64
	WorstRadiusEdge float64
	// Worst is the index of the triangle with the smallest angle.
	Worst int
	// Underflowed counts the triangles whose area rounded to zero, which is the number
	// to read before believing any of the areas above.
	Underflowed int
}

// Summarise returns the metrics of a whole mesh.
func Summarise(mesh *delaunay.Mesh) (Summary, error) {
	if mesh == nil || len(mesh.Triangles) == 0 {
		return Summary{}, fmt.Errorf("an empty mesh has no shape to summarise")
	}
	out := Summary{
		Triangles: len(mesh.Triangles),
		MinAngle:  math.Inf(1),
		MaxAngle:  math.Inf(-1),
		MinArea:   math.Inf(1),
		MaxArea:   math.Inf(-1),
		Worst:     -1,
	}
	total := 0.0
	for index := range mesh.Triangles {
		metrics, err := Of(mesh, index)
		if err != nil {
			return Summary{}, fmt.Errorf("triangle %d: %w", index, err)
		}
		if metrics.MinAngle < out.MinAngle {
			out.MinAngle = metrics.MinAngle
			out.Worst = index
		}
		out.MaxAngle = math.Max(out.MaxAngle, metrics.MaxAngle)
		out.MinArea = math.Min(out.MinArea, metrics.Area)
		out.MaxArea = math.Max(out.MaxArea, metrics.Area)
		out.WorstAspect = math.Max(out.WorstAspect, metrics.AspectRatio)
		out.WorstRadiusEdge = math.Max(out.WorstRadiusEdge, metrics.RadiusEdgeRatio)
		out.TotalArea += metrics.Area
		if metrics.AreaUnderflowed {
			out.Underflowed++
		}
		total += metrics.MinAngle
	}
	out.MeanMinAngle = total / float64(len(mesh.Triangles))
	return out, nil
}

// Describe renders the summary.
func (s Summary) Describe() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%d triangle(s), smallest angle %.4g degrees at triangle %d",
		s.Triangles, s.MinAngle, s.Worst)
	fmt.Fprintf(&builder, ", largest angle %.4g, mean smallest %.4g", s.MaxAngle, s.MeanMinAngle)
	fmt.Fprintf(&builder, ", worst aspect %.4g", s.WorstAspect)
	return builder.String()
}

// Histogram counts the triangles by their smallest angle, over buckets that divide the
// range from zero to sixty degrees. Sixty is the largest a smallest angle can be, and
// only an equilateral triangle reaches it.
func Histogram(mesh *delaunay.Mesh, buckets int) ([]int, error) {
	if buckets < 1 {
		return nil, fmt.Errorf("a histogram needs at least one bucket, got %d", buckets)
	}
	if mesh == nil || len(mesh.Triangles) == 0 {
		return nil, fmt.Errorf("an empty mesh has no histogram")
	}
	counts := make([]int, buckets)
	width := 60.0 / float64(buckets)
	for index := range mesh.Triangles {
		metrics, err := Of(mesh, index)
		if err != nil {
			return nil, fmt.Errorf("triangle %d: %w", index, err)
		}
		bucket := int(metrics.MinAngle / width)
		if bucket >= buckets {
			bucket = buckets - 1
		}
		if bucket < 0 {
			bucket = 0
		}
		counts[bucket]++
	}
	return counts, nil
}

// Slivers returns the triangles whose smallest angle is below a threshold in degrees,
// in mesh order.
func Slivers(mesh *delaunay.Mesh, threshold float64) ([]int, error) {
	if threshold <= 0 || threshold >= 60 {
		return nil, fmt.Errorf("a sliver threshold has to lie between 0 and 60 degrees, got %g", threshold)
	}
	if mesh == nil || len(mesh.Triangles) == 0 {
		return nil, fmt.Errorf("an empty mesh has no triangles to test")
	}
	out := []int{}
	for index := range mesh.Triangles {
		metrics, err := Of(mesh, index)
		if err != nil {
			return nil, fmt.Errorf("triangle %d: %w", index, err)
		}
		if metrics.MinAngle < threshold {
			out = append(out, index)
		}
	}
	return out, nil
}
