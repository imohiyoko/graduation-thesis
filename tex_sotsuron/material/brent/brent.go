// Copyright ©2025 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"math"
)

// Status represents the status of the optimization.
type Status int

const (
	NotTerminated Status = iota
	Converged
	IterationLimit
	Failure
)

// IterData stores the snapshot of each iteration for visualization.
type IterData struct {
	Iter   int
	A, B   float64 // Current interval boundaries
	X1     float64 // Best point (x)
	X2     float64 // Second best point (w)
	X3     float64 // Third best point (v)
	F1     float64 // f(x1)
	F2     float64 // f(x2)
	F3     float64 // f(x3)
	Method string  // "Parabolic" or "Golden"
}

// Settings allows fine-grained control over the optimization process.
type Settings struct {
	Tol           float64
	MaxIterations int
	Recorder      func(IterData)
}

// Result holds the result of the minimization.
type Result struct {
	X          float64
	F          float64
	Iterations int
	Status     Status
}

// BrentMin finds the local minimum of the scalar function f in the interval [min, max].
func BrentMin(f func(float64) float64, min, max float64, settings *Settings) (Result, error) {
	const (
		epsilon     = 0x1p-52
		sqrtEpsilon = 0x1p-26
		c           = 0.3819660112501051517954131656343618822796908201942371
	)

	tol := sqrtEpsilon
	maxIter := 100
	var recorder func(IterData)

	if settings != nil {
		if settings.Tol > 0 {
			tol = settings.Tol
		}
		if settings.MaxIterations > 0 {
			maxIter = settings.MaxIterations
		}
		recorder = settings.Recorder
	}

	a, b := min, max
	x := a + c*(b-a)
	w, v := x, x
	var e, d float64

	fx := f(x)
	fw, fv := fx, fx

	for i := 1; ; i++ {
		// Log current state before next step
		if recorder != nil {
			m := "Golden"
			if math.Abs(e) > (epsilon*math.Abs(x) + tol/3) {
				m = "SPI Attempt"
			}
			recorder(IterData{i, a, b, x, w, v, fx, fw, fv, m})
		}

		if i > maxIter {
			return Result{X: x, F: fx, Iterations: i, Status: IterationLimit}, nil
		}

		xm := (a + b) / 2
		tol1 := epsilon*math.Abs(x) + tol/3
		tol2 := 2 * tol1

		if math.Abs(x-xm) <= (tol2 - (b-a)/2) {
			return Result{X: x, F: fx, Iterations: i, Status: Converged}, nil
		}

		if math.Abs(e) > tol1 {
			r := (x - w) * (fx - fv)
			q := (x - v) * (fx - fw)
			p := (x-v)*q - (x-w)*r
			q = 2 * (q - r)
			if q > 0 {
				p = -p
			}
			q = math.Abs(q)
			etemp := e
			e = d

			if math.Abs(p) >= math.Abs(q*etemp/2) || p <= q*(a-x) || p >= q*(b-x) {
				if x >= xm {
					e = a - x
				} else {
					e = b - x
				}
				d = c * e
			} else {
				d = p / q
				u := x + d
				if u-a < tol2 || b-u < tol2 {
					if xm-x >= 0 {
						d = tol1
					} else {
						d = -tol1
					}
				}
			}
		} else {
			if x >= xm {
				e = a - x
			} else {
				e = b - x
			}
			d = c * e
		}

		u := 0.0
		if math.Abs(d) >= tol1 {
			u = x + d
		} else {
			if d > 0 {
				u = x + tol1
			} else {
				u = x - tol1
			}
		}

		fu := f(u)

		if fu <= fx {
			if u >= x {
				a = x
			} else {
				b = x
			}
			v, fv = w, fw
			w, fw = x, fx
			x, fx = u, fu
		} else {
			if u < x {
				a = u
			} else {
				b = u
			}
			if fu <= fw || w == x {
				v, fv = w, fw
				w, fw = u, fu
			} else if fu <= fv || v == x || v == w {
				v, fv = u, fu
			}
		}
	}
}
