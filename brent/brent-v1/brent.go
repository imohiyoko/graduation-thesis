package main

import (
	"fmt"
	"math"
)

// Machine epsilon for float64 (approx 2.22e-16)
// 1.0 + epsilon != 1.0 となる最小の正の数
const epsilon = 1.0 / (1 << 52)

// BrentMin finds the local minimum of the scalar function f in the interval [min, max].
// It uses Brent's method (combination of golden section search and successive parabolic interpolation).
// This is a Go translation of Netlib's fmin.f: http://www.netlib.org/opt/fmin.f
//
// Parameters:
//
//	f:   The function to minimize.
//	min: The lower bound of the interval.
//	max: The upper bound of the interval.
//	tol: The tolerance for the stopping criterion.
//	     If tol <= 0, a default value based on machine epsilon is used.
func BrentMin(f func(float64) float64, min, max, tol float64) (float64, error) {
	// c is the squared inverse of the golden ratio: (3 - sqrt(5))/2
	const c = 0.3819660112501051517954131656343618822796908201942371

	if tol <= 0 {
		tol = math.Sqrt(epsilon)
	}

	a := min
	b := max
	x := a + c*(b-a)
	w := x
	v := x
	e := 0.0 // step size of the step before last
	d := 0.0 // step size of the last step (moved to local scope)

	fx := f(x)
	fw := fx
	fv := fx

	// Main loop
	for {
		xm := 0.5 * (a + b)
		tol1 := epsilon*math.Abs(x) + tol/3.0
		tol2 := 2.0 * tol1

		// Check stopping criterion
		if math.Abs(x-xm) <= (tol2 - 0.5*(b-a)) {
			return x, nil
		}

		// Is golden-section necessary?
		if math.Abs(e) > tol1 {
			r := (x - w) * (fx - fv)
			q := (x - v) * (fx - fw)
			p := (x-v)*q - (x-w)*r
			q = 2.0 * (q - r)
			if q > 0 {
				p = -p
			}
			q = math.Abs(q)
			etemp := e
			e = d

			// Check acceptability of the parabolic fit
			if math.Abs(p) >= math.Abs(0.5*q*etemp) || p <= q*(a-x) || p >= q*(b-x) {
				if x >= xm {
					e = a - x
				} else {
					e = b - x
				}
				d = c * e
			} else {
				// Parabolic interpolation step accepted
				d = p / q
				u := x + d
				// Evaluate f must not be close to a or b
				if u-a < tol2 || b-u < tol2 {
					if xm-x >= 0 {
						d = tol1
					} else {
						d = -tol1
					}
				}
			}
		} else {
			// Golden-section step
			if x >= xm {
				e = a - x
			} else {
				e = b - x
			}
			d = c * e
		}

		// f must not be evaluated too close to x
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

		// Update a, b, v, w, and x
		if fu <= fx {
			if u >= x {
				a = x
			} else {
				b = x
			}
			v = w
			fv = fw
			w = x
			fw = fx
			x = u
			fx = fu
		} else {
			if u < x {
				a = u
			} else {
				b = u
			}
			if fu <= fw || w == x {
				v = w
				fv = fw
				w = u
				fw = fu
			} else if fu <= fv || v == x || v == w {
				v = u
				fv = fu
			}
		}
	}
}

func main() {
	fmt.Println("=== Brent's Method Implementation Test ===")

	f1 := func(x float64) float64 {
		return (x-2)*(x-2) + 1
	}

	fmt.Println("\nTest Case 1: f(x) = (x - 2)^2 + 1")
	minX, err := BrentMin(f1, 0, 5, 1e-8)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result x: %.15f\n", minX)
		fmt.Printf("Result f(x): %.15f\n", f1(minX))
		fmt.Printf("Expected x: 2.0\n")
		fmt.Printf("Difference: %.2e\n", math.Abs(minX-2.0))
	}

	// テストケース 2: f(x) = x * cos(x) (0 <= x <= 5)
	// 期待される最小値: x ≈ 3.425618...
	f2 := func(x float64) float64 {
		return x * math.Cos(x)
	}

	fmt.Println("\nTest Case 2: f(x) = x * cos(x)")
	expectedX2 := 3.425618459481728
	// tol=0 -> 自動的に sqrt(eps) ≈ 1.49e-8 が使用される
	minX2, err := BrentMin(f2, 0, 5, 0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result x: %.15f\n", minX2)
		fmt.Printf("Result f(x): %.15f\n", f2(minX2))
		fmt.Printf("Expected x: %.15f\n", expectedX2)
		fmt.Printf("Difference: %.2e\n", math.Abs(minX2-expectedX2))
	}
}
