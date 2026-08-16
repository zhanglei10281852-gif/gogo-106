// Package exact decides geometric predicates without ever guessing.
//
// Almost every mesh algorithm rests on two questions: which side of a line does a
// point fall on, and is a point inside the circle through three others. Answering
// them with plain floating point arithmetic is not merely inaccurate, it is
// inconsistent: the same three points can be reported as clockwise from one call site
// and counterclockwise from another, and an algorithm that believes both answers
// builds a mesh that contradicts itself.
//
// The way out is a filter. The determinant is evaluated in floating point together
// with a bound on how far that evaluation can be from the truth. If the result is
// larger than the bound its sign is certain and it is returned. If it is not, the
// same determinant is evaluated again over the rationals, where there is no rounding
// at all. Every float64 is a rational, so the second path is exact and the predicate
// as a whole never returns a sign it cannot justify.
package exact

import (
	"math"
	"math/big"
)

// epsilon is half the distance between one and the next representable number, which
// is the relative error of a single rounded operation.
const epsilon = 1.0 / (1 << 53)

// orientBound scales the sum of the two products of the orientation determinant into
// a bound on the error of their difference. The constant is deliberately larger than
// the tight bound: a bound that is too generous only sends more cases down the exact
// path, while a bound that is too small returns a sign that may be wrong.
const orientBound = 16 * epsilon

// circleBound is the matching constant for the in-circle determinant, whose
// evaluation involves more operations and so admits more error.
const circleBound = 64 * epsilon

// Sign values returned by the predicates.
const (
	Clockwise        = -1
	Collinear        = 0
	CounterClockwise = 1
)

// Stats records which path the predicates took.
type Stats struct {
	// Filtered counts the decisions the floating point filter settled on its own.
	Filtered int
	// Exact counts the decisions that had to be repeated over the rationals.
	Exact int
}

// Total is how many decisions were made.
func (s Stats) Total() int { return s.Filtered + s.Exact }

// Predicates answers the geometric questions and counts how it answered them, which
// is what lets a report show that the filter is doing its job.
type Predicates struct {
	stats Stats
}

// Stats returns the counters. A nil receiver has no counters and returns the zero
// value, so a caller that does not care can pass nil.
func (p *Predicates) Stats() Stats {
	if p == nil {
		return Stats{}
	}
	return p.stats
}

// Reset clears the counters.
func (p *Predicates) Reset() {
	if p == nil {
		return
	}
	p.stats = Stats{}
}

// record notes which path a decision took.
func (p *Predicates) record(exact bool) {
	if p == nil {
		return
	}
	if exact {
		p.stats.Exact++
		return
	}
	p.stats.Filtered++
}

// Orient2D reports whether c lies to the left of the directed line from a to b.
// The result is CounterClockwise, Clockwise or Collinear.
func (p *Predicates) Orient2D(ax, ay, bx, by, cx, cy float64) int {
	left := (ax - cx) * (by - cy)
	right := (ay - cy) * (bx - cx)
	determinant := left - right
	bound := orientBound * (math.Abs(left) + math.Abs(right))
	if determinant > bound || -determinant > bound {
		p.record(false)
		return sign(determinant)
	}
	p.record(true)
	return exactOrient2D(ax, ay, bx, by, cx, cy)
}

// InCircle reports whether d lies inside the circle through a, b and c, which must
// be given counterclockwise. The result is +1 inside, -1 outside and 0 on the circle.
func (p *Predicates) InCircle(ax, ay, bx, by, cx, cy, dx, dy float64) int {
	adx, ady := ax-dx, ay-dy
	bdx, bdy := bx-dx, by-dy
	cdx, cdy := cx-dx, cy-dy

	bdxcdy := bdx * cdy
	cdxbdy := cdx * bdy
	cdxady := cdx * ady
	adxcdy := adx * cdy
	adxbdy := adx * bdy
	bdxady := bdx * ady

	alift := adx*adx + ady*ady
	blift := bdx*bdx + bdy*bdy
	clift := cdx*cdx + cdy*cdy

	determinant := alift*(bdxcdy-cdxbdy) + blift*(cdxady-adxcdy) + clift*(adxbdy-bdxady)
	permanent := (math.Abs(bdxcdy)+math.Abs(cdxbdy))*alift +
		(math.Abs(cdxady)+math.Abs(adxcdy))*blift +
		(math.Abs(adxbdy)+math.Abs(bdxady))*clift
	bound := circleBound * permanent
	if determinant > bound || -determinant > bound {
		p.record(false)
		return sign(determinant)
	}
	p.record(true)
	return exactInCircle(ax, ay, bx, by, cx, cy, dx, dy)
}

// sign reduces a determinant to -1, 0 or 1.
func sign(value float64) int {
	switch {
	case value > 0:
		return 1
	case value < 0:
		return -1
	default:
		return 0
	}
}

// rat converts a float64 to the rational it exactly is.
func rat(value float64) *big.Rat {
	out := new(big.Rat)
	out.SetFloat64(value)
	return out
}

// subRat returns the exact difference of two float64 values.
func subRat(left, right float64) *big.Rat {
	return new(big.Rat).Sub(rat(left), rat(right))
}

// exactOrient2D evaluates the orientation determinant over the rationals.
func exactOrient2D(ax, ay, bx, by, cx, cy float64) int {
	acx := subRat(ax, cx)
	bcy := subRat(by, cy)
	acy := subRat(ay, cy)
	bcx := subRat(bx, cx)
	left := new(big.Rat).Mul(acx, bcy)
	right := new(big.Rat).Mul(acy, bcx)
	return left.Cmp(right)
}

// exactInCircle evaluates the in-circle determinant over the rationals.
func exactInCircle(ax, ay, bx, by, cx, cy, dx, dy float64) int {
	adx, ady := subRat(ax, dx), subRat(ay, dy)
	bdx, bdy := subRat(bx, dx), subRat(by, dy)
	cdx, cdy := subRat(cx, dx), subRat(cy, dy)

	alift := addProducts(adx, adx, ady, ady)
	blift := addProducts(bdx, bdx, bdy, bdy)
	clift := addProducts(cdx, cdx, cdy, cdy)

	first := subProducts(bdx, cdy, cdx, bdy)
	second := subProducts(cdx, ady, adx, cdy)
	third := subProducts(adx, bdy, bdx, ady)

	total := new(big.Rat).Mul(alift, first)
	total.Add(total, new(big.Rat).Mul(blift, second))
	total.Add(total, new(big.Rat).Mul(clift, third))
	return total.Sign()
}

// addProducts returns a*b + c*d exactly.
func addProducts(a, b, c, d *big.Rat) *big.Rat {
	out := new(big.Rat).Mul(a, b)
	return out.Add(out, new(big.Rat).Mul(c, d))
}

// subProducts returns a*b - c*d exactly.
func subProducts(a, b, c, d *big.Rat) *big.Rat {
	out := new(big.Rat).Mul(a, b)
	return out.Sub(out, new(big.Rat).Mul(c, d))
}

// Orient2D answers the orientation question without counting.
func Orient2D(ax, ay, bx, by, cx, cy float64) int {
	var predicates *Predicates
	return predicates.Orient2D(ax, ay, bx, by, cx, cy)
}

// InCircle answers the in-circle question without counting.
func InCircle(ax, ay, bx, by, cx, cy, dx, dy float64) int {
	var predicates *Predicates
	return predicates.InCircle(ax, ay, bx, by, cx, cy, dx, dy)
}

// ExactOrient2D answers the orientation question over the rationals, skipping the
// filter. It is the reference the filtered path is checked against.
func ExactOrient2D(ax, ay, bx, by, cx, cy float64) int {
	return exactOrient2D(ax, ay, bx, by, cx, cy)
}

// ExactInCircle answers the in-circle question over the rationals, skipping the
// filter.
func ExactInCircle(ax, ay, bx, by, cx, cy, dx, dy float64) int {
	return exactInCircle(ax, ay, bx, by, cx, cy, dx, dy)
}

// NaiveOrient2D answers the orientation question with plain floating point
// arithmetic and no filter at all, which is what the filtered path is compared
// against when a report needs to show that the difference matters.
func NaiveOrient2D(ax, ay, bx, by, cx, cy float64) int {
	return sign((ax-cx)*(by-cy) - (ay-cy)*(bx-cx))
}
