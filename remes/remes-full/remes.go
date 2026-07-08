package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

// --- Configuration Parameters ---
const (
	TargetDegree   = 20 // High degree to create fine ripples
	IntervalStart  = 0.5
	IntervalEnd    = 5.0
	MaxIterations  = 50    // More iterations for high degree
	ConvergenceTol = 1e-13 // Strict convergence

	// Grid sizes
	// 500 is enough to bracket the 22 extrema of a degree 20 polynomial,
	// but too coarse to pin-point them exactly. This highlights Brent's value.
	DenseGridSize     = 500
	SparseGridSize    = 500
	ValidationSamples = 10000000 // 10 million points for true error checking
)

// TargetFunction: f(x) = log(x)
func TargetFunction(x float64) float64 {
	return math.Log(x)
}

// OptimizationResult holds the outcome
type OptimizationResult struct {
	PhaseName        string
	Duration         time.Duration
	Iterations       int
	Converged        bool
	FinalDiffNorm    float64
	ReportedMaxError float64
	Coefficients     []float64
	ErrorHistory     []float64
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	results := []OptimizationResult{}

	fmt.Println("=== Remes Algorithm: Degree 20 Benchmark ===")
	fmt.Printf("CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("Target: log(x), Degree: %d, Interval: [%.1f, %.1f]\n", TargetDegree, IntervalStart, IntervalEnd)
	fmt.Printf("Convergence Criteria: ||P_{n+1} - P_n|| < %.1e\n", ConvergenceTol)
	fmt.Printf("Search Grid: %d points | Validation Grid: %d points\n", DenseGridSize, ValidationSamples)
	fmt.Println("-------------------------------------------------------------------------------")

	// --- Phase 1: Basic Grid Search ---
	res1 := runRemes("Phase1_Basic", DenseGridSize, findExtremaBasicGrid)
	results = append(results, res1)

	// --- Phase 2: Parallel Grid Search ---
	res2 := runRemes("Phase2_Parallel", DenseGridSize, findExtremaParallelGrid)
	results = append(results, res2)

	// --- Phase 3: Brent's Method (Improved Logic) ---
	res3 := runRemes("Phase3_Brent", SparseGridSize, findExtremaBrent)
	results = append(results, res3)

	printSummary(results)
	outputComparisonPlot(results)
}

// ---------------------------------------------------------
// Core Remes Engine
// ---------------------------------------------------------

type ExtremaFinderFunc func(errFunc func(float64) float64, degree, gridSize int, start, end float64) []float64

func runRemes(phaseName string, gridSize int, finder ExtremaFinderFunc) OptimizationResult {
	fmt.Printf("Running %-20s ... ", phaseName)
	startTime := time.Now()

	nodes := initializeNodes(TargetDegree, IntervalStart, IntervalEnd)
	var coefficients []float64
	var prevCoefficients []float64
	var errorHistory []float64
	var diffNorm float64
	var maxErr float64

	converged := false
	iter := 0

	for ; iter < MaxIterations; iter++ {
		// 1. Solve Linear System
		coeffs, _ := solveLinearSystem(nodes, TargetDegree, IntervalStart, IntervalEnd)
		if coeffs == nil {
			fmt.Println(" [Solver Error]")
			break
		}
		coefficients = coeffs

		// 2. Check Convergence: ||P_{n} - P_{n-1}||
		if iter > 0 {
			diffFunc := func(x float64) float64 {
				pCurr := evaluatePolynomial(coefficients, x, IntervalStart, IntervalEnd)
				pPrev := evaluatePolynomial(prevCoefficients, x, IntervalStart, IntervalEnd)
				return math.Abs(pCurr - pPrev)
			}
			diffNorm = parallelMaxNorm(diffFunc, IntervalStart, IntervalEnd, ValidationSamples)

			if diffNorm < ConvergenceTol {
				converged = true
			}
		}
		prevCoefficients = make([]float64, len(coefficients))
		copy(prevCoefficients, coefficients)

		// 3. Define Error Function
		errFunc := createErrorFunction(coefficients, IntervalStart, IntervalEnd)

		// 4. Find Extrema
		extrema := finder(errFunc, TargetDegree, gridSize, IntervalStart, IntervalEnd)

		// 5. Calculate Max Error on dense validation grid
		absErrFunc := func(x float64) float64 { return math.Abs(errFunc(x)) }
		maxErr = parallelMaxNorm(absErrFunc, IntervalStart, IntervalEnd, ValidationSamples)
		errorHistory = append(errorHistory, maxErr)

		if converged {
			break
		}
		nodes = extrema
	}

	duration := time.Since(startTime)
	fmt.Printf("Done (%v) Iter: %d, DiffNorm: %.2e, MaxErr: %.2e\n", duration, iter+1, diffNorm, maxErr)

	outputResultFile(phaseName, coefficients, iter+1, duration, maxErr, diffNorm)

	return OptimizationResult{
		PhaseName:        phaseName,
		Duration:         duration,
		Iterations:       iter + 1,
		Converged:        converged,
		FinalDiffNorm:    diffNorm,
		ReportedMaxError: maxErr,
		Coefficients:     coefficients,
		ErrorHistory:     errorHistory,
	}
}

// ---------------------------------------------------------
// Parallel Evaluation
// ---------------------------------------------------------

func parallelMaxNorm(f func(float64) float64, start, end float64, samples int) float64 {
	numCPU := runtime.NumCPU()
	if samples < numCPU {
		samples = numCPU
	}
	var wg sync.WaitGroup
	resultCh := make(chan float64, numCPU)
	samplesPerChunk := samples / numCPU
	stepSize := (end - start) / float64(samples)

	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		chunkStartIdx := i * samplesPerChunk
		chunkEndIdx := (i + 1) * samplesPerChunk
		if i == numCPU-1 {
			chunkEndIdx = samples
		}
		go func(sIdx, eIdx int) {
			defer wg.Done()
			localMax := -1.0
			for j := sIdx; j <= eIdx; j++ {
				x := start + float64(j)*stepSize
				if x > end {
					x = end
				}
				val := f(x)
				if val > localMax {
					localMax = val
				}
			}
			resultCh <- localMax
		}(chunkStartIdx, chunkEndIdx)
	}
	wg.Wait()
	close(resultCh)

	globalMax := 0.0
	for max := range resultCh {
		if max > globalMax {
			globalMax = max
		}
	}
	return globalMax
}

// ---------------------------------------------------------
// Extrema Search Implementations
// ---------------------------------------------------------

// Phase 1: Basic Grid
func findExtremaBasicGrid(errFunc func(float64) float64, degree, gridSize int, start, end float64) []float64 {
	stepSize := (end - start) / float64(gridSize)
	var extrema []float64
	extrema = append(extrema, start)
	prevVal := math.Abs(errFunc(start))
	currVal := math.Abs(errFunc(start + stepSize))

	for i := 1; i < gridSize; i++ {
		x := start + float64(i)*stepSize
		nextVal := math.Abs(errFunc(x + stepSize))
		if currVal >= prevVal && currVal >= nextVal {
			extrema = append(extrema, x)
		}
		prevVal = currVal
		currVal = nextVal
	}
	extrema = append(extrema, end)
	return selectBestExtrema(extrema, errFunc, degree+2)
}

// Phase 2: Parallel Grid Search
func findExtremaParallelGrid(errFunc func(float64) float64, degree, gridSize int, start, end float64) []float64 {
	numCPU := runtime.NumCPU()
	var wg sync.WaitGroup
	resultCh := make(chan []float64, numCPU)
	chunkSize := gridSize / numCPU
	stepSize := (end - start) / float64(gridSize)

	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		startIdx := i * chunkSize
		endIdx := (i + 1) * chunkSize
		if i == numCPU-1 {
			endIdx = gridSize
		}
		go func(s, e int) {
			defer wg.Done()
			var local []float64
			for j := s; j < e; j++ {
				if j == 0 || j >= gridSize-1 {
					continue
				}
				x := start + float64(j)*stepSize
				val := math.Abs(errFunc(x))
				prev := math.Abs(errFunc(x - stepSize))
				next := math.Abs(errFunc(x + stepSize))
				if val >= prev && val >= next {
					local = append(local, x)
				}
			}
			resultCh <- local
		}(startIdx, endIdx)
	}
	wg.Wait()
	close(resultCh)
	var all []float64
	all = append(all, start)
	for res := range resultCh {
		all = append(all, res...)
	}
	all = append(all, end)
	return selectBestExtrema(all, errFunc, degree+2)
}

// Phase 3: Brent's Method (Improved)
func findExtremaBrent(errFunc func(float64) float64, degree, gridSize int, start, end float64) []float64 {
	// 1. Coarse Grid Search to find brackets
	candidates := findExtremaBasicGrid(errFunc, degree, gridSize, start, end)
	var refined []float64
	stepSize := (end - start) / float64(gridSize)
	brentTol := 1e-10

	for _, x := range candidates {
		// If very close to boundaries, keep as is (Brent might be unstable exactly at edge)
		if math.Abs(x-start) < 1e-9 || math.Abs(x-end) < 1e-9 {
			refined = append(refined, x)
			continue
		}

		// Define bracket [minX, maxX]
		minX, maxX := x-stepSize, x+stepSize
		if minX < start {
			minX = start
		}
		if maxX > end {
			maxX = end
		}

		y := errFunc(x)
		var targetFunc func(float64) float64

		// Improved Logic:
		// The error function E(x) oscillates between positive and negative values.
		// If y > 0, we are near a positive peak -> Maximize y -> Minimize -y
		// If y < 0, we are near a negative peak -> Minimize y
		if y > 0 {
			targetFunc = func(t float64) float64 { return -errFunc(t) }
		} else {
			targetFunc = errFunc
		}

		resX, _ := brentMinimize(targetFunc, minX, maxX, brentTol)
		refined = append(refined, resX)
	}
	return selectBestExtrema(refined, errFunc, degree+2)
}

// ---------------------------------------------------------
// Numerical Utilities
// ---------------------------------------------------------

func solveLinearSystem(nodes []float64, degree int, start, end float64) ([]float64, float64) {
	m := degree + 2
	A := mat.NewDense(m, m, nil)
	bVec := make([]float64, m)
	for i := 0; i < m; i++ {
		xi := nodes[i]
		bVec[i] = TargetFunction(xi)
		ui := (2.0*xi - (start + end)) / (end - start)
		tPrev2, tPrev1 := 0.0, 1.0
		A.Set(i, 0, 1.0)
		if degree >= 1 {
			tCurr := ui
			A.Set(i, 1, tCurr)
			tPrev2, tPrev1 = tPrev1, tCurr
		}
		for j := 2; j <= degree; j++ {
			tCurr := 2*ui*tPrev1 - tPrev2
			A.Set(i, j, tCurr)
			tPrev2, tPrev1 = tPrev1, tCurr
		}
		A.Set(i, degree+1, -math.Pow(-1, float64(i)))
	}
	vecB := mat.NewVecDense(m, bVec)
	var x mat.VecDense
	if err := x.SolveVec(A, vecB); err != nil {
		return nil, 0.0
	}
	sol := x.RawVector().Data
	return sol[:degree+1], sol[degree+1]
}

func initializeNodes(degree int, start, end float64) []float64 {
	nodes := make([]float64, degree+2)
	for k := 0; k < degree+2; k++ {
		nodes[k] = (start+end)/2.0 + (end-start)/2.0*math.Cos(float64(k)*math.Pi/float64(degree+1))
	}
	sort.Float64s(nodes)
	return nodes
}

func evaluatePolynomial(coeffs []float64, x, start, end float64) float64 {
	u := (2.0*x - (start + end)) / (end - start)
	sum := 0.0
	tPrev2, tPrev1 := 0.0, 1.0
	sum += coeffs[0] * tPrev1
	if len(coeffs) > 1 {
		tCurr := u
		sum += coeffs[1] * tCurr
		tPrev2, tPrev1 = tPrev1, tCurr
		for j := 2; j < len(coeffs); j++ {
			tCurr = 2*u*tPrev1 - tPrev2
			sum += coeffs[j] * tCurr
			tPrev2, tPrev1 = tPrev1, tCurr
		}
	}
	return sum
}

func createErrorFunction(coeffs []float64, start, end float64) func(float64) float64 {
	return func(x float64) float64 {
		return evaluatePolynomial(coeffs, x, start, end) - TargetFunction(x)
	}
}

func selectBestExtrema(candidates []float64, errFunc func(float64) float64, count int) []float64 {
	sort.Float64s(candidates)
	unique := candidates[:0]
	if len(candidates) > 0 {
		unique = append(unique, candidates[0])
		for i := 1; i < len(candidates); i++ {
			if math.Abs(candidates[i]-unique[len(unique)-1]) > 1e-6 {
				unique = append(unique, candidates[i])
			}
		}
	}
	type point struct {
		x, err float64
	}
	points := make([]point, len(unique))
	for i, x := range unique {
		points[i] = point{x, math.Abs(errFunc(x))}
	}
	sort.Slice(points, func(i, j int) bool {
		return points[i].err > points[j].err
	})
	if len(points) < count {
		count = len(points)
	}
	final := make([]float64, count)
	for i := 0; i < count; i++ {
		final[i] = points[i].x
	}
	sort.Float64s(final)
	return final
}

func brentMinimize(f func(float64) float64, a, b, tol float64) (float64, int) {
	const goldenRatio = 0.3819660112501051
	x := a + goldenRatio*(b-a)
	w, v := x, x
	e := 0.0
	fx := f(x)
	fw, fv := fx, fx
	d := 0.0
	iter := 0
	for ; iter < 100; iter++ {
		mid := (a + b) / 2.0
		tol1 := tol*math.Abs(x) + tol
		tol2 := 2.0 * tol1
		if math.Abs(x-mid) <= (tol2 - 0.5*(b-a)) {
			break
		}
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
			if math.Abs(p) >= math.Abs(0.5*q*etemp) || p <= q*(a-x) || p >= q*(b-x) {
				if x >= mid {
					e = a - x
				} else {
					e = b - x
				}
				d = goldenRatio * e
			} else {
				d = p / q
				u := x + d
				if u-a < tol2 || b-u < tol2 {
					d = math.Copysign(tol1, mid-x)
				}
			}
		} else {
			if x >= mid {
				e = a - x
			} else {
				e = b - x
			}
			d = goldenRatio * e
		}
		u := 0.0
		if math.Abs(d) >= tol1 {
			u = x + d
		} else {
			u = x + math.Copysign(tol1, d)
		}
		fu := f(u)
		if fu <= fx {
			if u >= x {
				a = x
			} else {
				b = x
			}
			v, w, x = w, x, u
			fv, fw, fx = fw, fx, fu
		} else {
			if u < x {
				a = u
			} else {
				b = u
			}
			if fu <= fw || w == x {
				v, w = w, u
				fv, fw = fw, fu
			} else if fu <= fv || v == x || v == w {
				v, fv = u, fu
			}
		}
	}
	return x, iter
}

// ---------------------------------------------------------
// Output Logic
// ---------------------------------------------------------

func outputResultFile(phaseName string, coeffs []float64, iter int, dur time.Duration, maxErr, diffNorm float64) {
	f, err := os.Create(fmt.Sprintf("result_%s.txt", phaseName))
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "Phase: %s\n", phaseName)
	fmt.Fprintf(f, "Execution Time: %v\n", dur)
	fmt.Fprintf(f, "Total Iterations: %d\n", iter)
	fmt.Fprintf(f, "Max Error Norm:    %.5e\n", maxErr)
	fmt.Fprintf(f, "Diff Norm (Pn-Pn-1): %.5e\n", diffNorm)
	fmt.Fprintf(f, "\nCoefficients:\n")
	for i, c := range coeffs {
		fmt.Fprintf(f, "T%d: %.10f\n", i, c)
	}
}

func outputComparisonPlot(results []OptimizationResult) {
	p := plot.New()
	p.Title.Text = "Remes Convergence: Max Error vs Iteration"
	p.X.Label.Text = "Iteration"
	p.Y.Label.Text = "Max Error (Log Scale)"
	p.Y.Scale = plot.LogScale{}
	p.Y.Tick.Marker = plot.LogTicks{}

	for i, res := range results {
		pts := make(plotter.XYs, len(res.ErrorHistory))
		for j, errVal := range res.ErrorHistory {
			pts[j].X = float64(j + 1)
			pts[j].Y = errVal
		}
		line, points, _ := plotter.NewLinePoints(pts)
		color := plotutil.Color(i)
		line.Color = color
		points.Color = color
		points.Shape = plotutil.Shape(i)
		p.Add(line, points)
		p.Legend.Add(res.PhaseName, line, points)
	}
	p.Add(plotter.NewGrid())
	p.Save(6*vg.Inch, 4*vg.Inch, "convergence_comparison.png")
}

func printSummary(results []OptimizationResult) {
	fmt.Println("\n=== Final Summary ===")
	fmt.Printf("%-20s | %-12s | %-4s | %-12s | %-12s\n", "Phase", "Time", "Iter", "Max Error", "Diff Norm")
	fmt.Println(strings.Repeat("-", 85))
	for _, r := range results {
		fmt.Printf("%-20s | %-12s | %-4d | %.5e  | %.5e\n",
			r.PhaseName, r.Duration, r.Iterations, r.ReportedMaxError, r.FinalDiffNorm)
	}
}
