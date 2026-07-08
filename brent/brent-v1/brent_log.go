package main

import (
	"fmt"
	"math"
	"os"
)

// Machine epsilon for float64 (approx 2.22e-16)
const epsilon = 1.0 / (1 << 52)

// LogEntry stores information for each iteration
type LogEntry struct {
	Iter int
	X    float64
	U    float64
	V    float64
	W    float64
	Min  float64
	Max  float64
}

// BrentMinLog finds the local minimum and logs each iteration
func BrentMinLog(f func(float64) float64, min, max, tol float64) (float64, []LogEntry, error) {
	const c = 0.3819660112501051517954131656343618822796908201942371

	if tol <= 0 {
		tol = math.Sqrt(epsilon)
	}

	var logs []LogEntry
	iter := 0

	a := min
	b := max
	x := a + c*(b-a)
	w := x
	v := x
	e := 0.0
	d := 0.0

	fx := f(x)
	fw := fx
	fv := fx

	// Log initial state
	logs = append(logs, LogEntry{
		Iter: iter,
		X:    x,
		U:    x, // 初回はuがまだないのでxと同じ
		V:    v,
		W:    w,
		Min:  a,
		Max:  b,
	})

	// Main loop
	for {
		iter++
		xm := 0.5 * (a + b)
		tol1 := epsilon*math.Abs(x) + tol/3.0
		tol2 := 2.0 * tol1

		// Check stopping criterion
		if math.Abs(x-xm) <= (tol2 - 0.5*(b-a)) {
			return x, logs, nil
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

		// Log before update
		logs = append(logs, LogEntry{
			Iter: iter,
			X:    x,
			U:    u,
			V:    v,
			W:    w,
			Min:  a,
			Max:  b,
		})

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
	fmt.Println("=== Brent's Method with Logging ===\n")

	// Test Case: f(x) = (x - 2)^2 + 1
	f := func(x float64) float64 {
		return (x-2)*(x-2) + 1
	}

	fmt.Println("f(x) = (x - 2)^2 + 1, [0, 5], tol=1e-8\n")
	minX, logs, err := BrentMinLog(f, 0, 5, 1e-8)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Minimum found at x = %.15f\n", minX)
	fmt.Printf("f(x) = %.15f\n\n", f(minX))

	// Print table for LaTeX
	fmt.Println("LaTeX Table Format:")
	fmt.Println("\\begin{table}[H]")
	fmt.Println("  \\centering")
	fmt.Println("  \\caption{Brent法の反復過程}")
	fmt.Println("  \\begin{tabular}{|r|c|c|c|c|c|c|}")
	fmt.Println("    \\hline")
	fmt.Println("    反復回数 & $x$ (最良) & $w$ (2番目) & $v$ (3番目) & $u$ (新規) & $[a, b]$ の $a$ & $[a, b]$ の $b$ \\\\")
	fmt.Println("    \\hline")

	for _, log := range logs {
		if log.Iter <= 19 { // 最初の20回まで出力
			fmt.Printf("    %2d & %.15f & %.15f & %.15f & %.15f & %.15f & %.15f \\\\\n",
				log.Iter, log.X, log.W, log.V, log.U, log.Min, log.Max)
		}
	}

	fmt.Println("    \\hline")
	fmt.Println("  \\end{tabular}")
	fmt.Println("\\end{table}")

	// Write to log.md file in Markdown format
	file, err := os.Create("brent_log.md")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	file.WriteString("# Brent's Method Log\n\n")
	file.WriteString("## 問題設定\n\n")
	file.WriteString("- 関数: $f(x) = (x - 2)^2 + 1$\n")
	file.WriteString("- 探索区間: $[0, 5]$\n")
	file.WriteString("- 許容誤差: $10^{-8}$\n\n")
	file.WriteString("## 結果\n\n")
	file.WriteString(fmt.Sprintf("- 最小値: $x = %.15f$\n", minX))
	file.WriteString(fmt.Sprintf("- $f(x) = %.15f$\n\n", f(minX)))

	file.WriteString("## 反復過程\n\n")
	file.WriteString("変数の意味：\n\n")
	file.WriteString("- $x$: 現在の最良解（最小の関数値を持つ点）\n")
	file.WriteString("- $w$: 2番目に良い点\n")
	file.WriteString("- $v$: 3番目に良い点\n")
	file.WriteString("- $u$: 今回の反復で新たに評価する点\n\n")
	file.WriteString("| 反復回数 | x (最良) | w (2番目) | v (3番目) | u (新規) | [a, b] の a | [a, b] の b |\n")
	file.WriteString("|:--------:|:--------:|:--------:|:--------:|:--------:|:-----------:|:-----------:|\n")
	for _, log := range logs {
		file.WriteString(fmt.Sprintf("| %2d | %.15f | %.15f | %.15f | %.15f | %.15f | %.15f |\n",
			log.Iter, log.X, log.W, log.V, log.U, log.Min, log.Max))
	}

	fmt.Println("\nLog file 'brent_log.md' has been created successfully.")

	// Write to log.txt file in plain text format
	txtFile, err := os.Create("brent_log.txt")
	if err != nil {
		fmt.Printf("Error creating txt file: %v\n", err)
		return
	}
	defer txtFile.Close()

	txtFile.WriteString("=== Brent's Method Log ===\n\n")
	txtFile.WriteString("問題設定\n")
	txtFile.WriteString("--------\n")
	txtFile.WriteString("関数: f(x) = (x - 2)^2 + 1\n")
	txtFile.WriteString("探索区間: [0, 5]\n")
	txtFile.WriteString("許容誤差: 1e-8\n\n")
	txtFile.WriteString("結果\n")
	txtFile.WriteString("----\n")
	txtFile.WriteString(fmt.Sprintf("最小値: x = %.15f\n", minX))
	txtFile.WriteString(fmt.Sprintf("f(x) = %.15f\n\n", f(minX)))

	txtFile.WriteString("反復過程\n")
	txtFile.WriteString("--------\n")
	txtFile.WriteString("変数の意味：\n")
	txtFile.WriteString("  x: 現在の最良解（最小の関数値を持つ点）\n")
	txtFile.WriteString("  w: 2番目に良い点\n")
	txtFile.WriteString("  v: 3番目に良い点\n")
	txtFile.WriteString("  u: 今回の反復で新たに評価する点\n\n")

	txtFile.WriteString("反復  x (最良)               w (2番目)              v (3番目)              u (新規)               [a, b] の a            [a, b] の b\n")
	txtFile.WriteString("====================================================================================================================================\n")
	for _, log := range logs {
		txtFile.WriteString(fmt.Sprintf("%2d    %.15f    %.15f    %.15f    %.15f    %.15f    %.15f\n",
			log.Iter, log.X, log.W, log.V, log.U, log.Min, log.Max))
	}

	fmt.Println("Log file 'brent_log.txt' has been created successfully.")
}
