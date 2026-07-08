package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"sort"
	"strings"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw" // マーカー描画用
)

// main はプログラムの開始点です．
func main() {
	fmt.Println("Remesアルゴリズム（正規化チェビシェフ基底 + Brent法）: log(x)の近似")
	remesAlgorithm()
}

// ターゲット関数 f(x)
func targetFunction(x float64) float64 {
	return 1.0 / (1.0 + 25.0*x*x)
}

// BrentStep: Brent法の1ステップの情報を保持する構造体
type BrentStep struct {
	Iter      int
	Method    string  // "SPI" or "Golden"
	X         float64 // 現在の最良点
	Fx        float64 // その時の関数値
	IntervalW float64 // 区間幅 (b - a)
}

// remesAlgorithm: 全体の制御
func remesAlgorithm() {
	// --- パラメータ設定 ---
	n := 18 // 多項式の次数
	intervalStart, intervalEnd := -1.0, 1.0
	gridSize := 1000        // グリッドサーチの分割数（Brent法の初期値探し用）
	maxLoop := 100          // 最大反復回数
	tolerance := 1e-15      // 収束判定の許容誤差
	brentTolerance := 1e-10 // Brent法の許容誤差

	// ログファイルの作成
	logFile, err := os.Create("remes_log.txt")
	if err != nil {
		log.Fatal("ログファイルの作成に失敗しました:", err)
	}
	defer logFile.Close()
	// ログ出力用ヘルパー関数
	writeLog := func(format string, a ...interface{}) {
		msg := fmt.Sprintf(format, a...)
		fmt.Print(msg)           // コンソールへ
		logFile.WriteString(msg) // ファイルへ
	}

	writeLog("Remesアルゴリズム開始: n=%d, 区間=[%.2f, %.2f]\n", n, intervalStart, intervalEnd)

	// 初期節点（チェビシェフ節点）
	nodes := initializeNodes(n, intervalStart, intervalEnd)

	var coeffs []float64
	var e float64
	prevMaxErr := math.Inf(1)
	var maxErrors []float64
	var errorSpreads []float64   // 誤差の最大値と最小値の差の推移
	var nodesHistory [][]float64 // 参照点の推移履歴

	for i := 0; i < maxLoop; i++ {
		writeLog("\n--- イテレーション %d ---\n", i+1)
		writeLog("現在の参照点: %s\n", formatFloatSlice(nodes, 5))
		nodesHistory = append(nodesHistory, make([]float64, len(nodes)))
		copy(nodesHistory[len(nodesHistory)-1], nodes)

		// 1. 連立方程式を解く（正規化チェビシェフ基底を使用）
		coeffs, e = solveLinearSystem(nodes, n, intervalStart, intervalEnd)
		if coeffs == nil {
			writeLog("計算を中断します．\n")
			return
		}

		writeLog("係数(Chebyshev): %s\n", formatFloatSlice(coeffs, 6))
		writeLog("均等誤差パラメータ E: %.5e\n", e)

		// 2. 誤差関数を定義（区間情報をクロージャに含める）
		eFunc := errorFunction(coeffs, intervalStart, intervalEnd)

		// 3. 極値探索（グリッドサーチ + Brent法）
		// 極値探索と同時に，Brent法の履歴データを受け取る
		extrema, brentHistories := findExtrema(eFunc, n, gridSize, intervalStart, intervalEnd, brentTolerance, writeLog)
		writeLog("極値点候補: %s\n", formatFloatSlice(extrema, 5))

		// ★Brent法の収束履歴グラフ作成 (SPI=○, Golden=△)
		if len(brentHistories) > 0 {
			if err := outputBrentConvergenceBatchPlot(brentHistories, i+1); err != nil {
				log.Printf("Brent収束プロット保存エラー: %v", err)
			}
		}

		// 4. グラフ描画（イテレーションごと）
		if err := outputIterationPlots(coeffs, i+1, intervalStart, intervalEnd, 1000); err != nil {
			log.Printf("プロット保存エラー: %v", err)
		}

		// 5. 収束判定情報の計算
		currentMaxErr := 0.0
		currentMinErr := math.Inf(1)
		for _, x := range extrema {
			absErr := math.Abs(eFunc(x))
			if absErr > currentMaxErr {
				currentMaxErr = absErr
			}
			if absErr < currentMinErr {
				currentMinErr = absErr
			}
		}

		spread := currentMaxErr - currentMinErr // 誤差のばらつき（交替定理からの乖離度）
		maxErrors = append(maxErrors, currentMaxErr)
		errorSpreads = append(errorSpreads, spread)

		writeLog("最大誤差: %.5e, 最小誤差: %.5e, ばらつき(幅): %.5e\n", currentMaxErr, currentMinErr, spread)

		// 収束判定（前回との最大誤差の差分）
		diff := 0.0
		if !math.IsInf(prevMaxErr, 1) {
			diff = math.Abs(currentMaxErr - prevMaxErr)
		}

		converged := false
		if !math.IsInf(prevMaxErr, 1) && diff < tolerance {
			converged = true
		}

		if converged {
			writeLog("\n収束しました（最大誤差差分が許容範囲内: %.2e < %.2e）．\n", diff, tolerance)
			// 最終的なLaTeX形式を出力
			latex := formatPolynomialLatex(coeffs, fmt.Sprintf("P_{%d}", n), "u", 10, 1e-14, true)
			writeLog("\n--- Result LaTeX ---\n")
			writeLog("%s\n", latex)
			writeLog("u = (2x - %.2f) / %.2f\n", intervalStart+intervalEnd, intervalEnd-intervalStart)
			break
		}

		prevMaxErr = currentMaxErr
		nodes = extrema // 次の参照点として更新

		if i == maxLoop-1 {
			writeLog("\n最大反復回数に到達（未収束）．\n")
		}
	}

	// 各種履歴のプロット保存
	if len(maxErrors) > 0 {
		if err := outputConvergencePlot(maxErrors); err != nil {
			log.Printf("収束履歴プロット保存エラー: %v", err)
		}
		if err := outputSpreadPlot(errorSpreads); err != nil {
			log.Printf("誤差ばらつきプロット保存エラー: %v", err)
		}
		if err := outputNodesHistoryPlot(nodesHistory); err != nil {
			log.Printf("参照点推移プロット保存エラー: %v", err)
		}
	}
	fmt.Println("ログを remes_log.txt に保存しました．")
}

// --- 数値計算コア部分: 正規化チェビシェフ基底 ---

func solveLinearSystem(nodes []float64, n int, a, b float64) (coeffs []float64, e float64) {
	m := n + 2
	if len(nodes) != m {
		log.Printf("Error: nodes count mismatch")
		return nil, 0.0
	}

	A := mat.NewDense(m, m, nil)
	bData := make([]float64, m)

	for i := 0; i < m; i++ {
		xi := nodes[i]
		bData[i] = targetFunction(xi)

		ui := (2.0*xi - (a + b)) / (b - a)

		t_prev2 := 0.0
		t_prev1 := 1.0

		A.Set(i, 0, 1.0)
		if n >= 1 {
			t_curr := ui
			A.Set(i, 1, t_curr)
			t_prev2 = t_prev1
			t_prev1 = t_curr
		}
		for j := 2; j <= n; j++ {
			t_curr := 2*ui*t_prev1 - t_prev2
			A.Set(i, j, t_curr)
			t_prev2 = t_prev1
			t_prev1 = t_curr
		}

		A.Set(i, n+1, -math.Pow(-1, float64(i)))
	}

	vecB := mat.NewVecDense(m, bData)
	var c mat.VecDense
	if err := c.SolveVec(A, vecB); err != nil {
		log.Printf("連立方程式の解決失敗: %v", err)
		return nil, 0.0
	}

	sol := c.RawVector().Data
	return sol[:n+1], sol[n+1]
}

// polynomial: 正規化チェビシェフ基底による多項式評価
func polynomial(coeffs []float64, x, a, b float64) float64 {
	if len(coeffs) == 0 {
		return 0.0
	}
	u := (2.0*x - (a + b)) / (b - a)

	sum := 0.0
	t_prev2 := 0.0
	t_prev1 := 1.0

	sum += coeffs[0] * t_prev1
	if len(coeffs) > 1 {
		t_curr := u
		sum += coeffs[1] * t_curr
		t_prev2 = t_prev1
		t_prev1 = t_curr

		for j := 2; j < len(coeffs); j++ {
			t_curr = 2*u*t_prev1 - t_prev2
			sum += coeffs[j] * t_curr
			t_prev2 = t_prev1
			t_prev1 = t_curr
		}
	}
	return sum
}

// errorFunction: 誤差関数 E(x) を返すクロージャ
func errorFunction(coeffs []float64, a, b float64) func(float64) float64 {
	return func(x float64) float64 {
		return polynomial(coeffs, x, a, b) - targetFunction(x)
	}
}

// findExtrema: 極値探索と履歴データを返す
func findExtrema(eFunc func(float64) float64, n int, gridSize int, a, b, brentTol float64, logger func(string, ...interface{})) ([]float64, map[int][]BrentStep) {
	logger("ステップ: 極値点探索 (Grid Search + Brent's Method)\n")
	var extrema []float64
	histories := make(map[int][]BrentStep)

	h := (b - a) / float64(gridSize)

	y_prev := eFunc(a)
	y_curr := eFunc(a + h)

	extrema = append(extrema, a)

	extremumID := 0 // 極値IDカウンタ

	for i := 1; i < gridSize-1; i++ {
		x := a + float64(i)*h
		y_next := eFunc(x + h)

		searchMin := x - h
		searchMax := x + h

		// 極大 (山)
		if y_curr > y_prev && y_curr > y_next {
			extremumID++
			negFunc := func(t float64) float64 { return -eFunc(t) }
			refinedX, hist := brentMinimize(negFunc, searchMin, searchMax, brentTol, logger, extremumID)
			extrema = append(extrema, refinedX)
			histories[extremumID] = hist
		}
		// 極小 (谷)
		if y_curr < y_prev && y_curr < y_next {
			extremumID++
			refinedX, hist := brentMinimize(eFunc, searchMin, searchMax, brentTol, logger, extremumID)
			extrema = append(extrema, refinedX)
			histories[extremumID] = hist
		}
		y_prev = y_curr
		y_curr = y_next
	}
	extrema = append(extrema, b)

	sort.Slice(extrema, func(i, j int) bool {
		return math.Abs(eFunc(extrema[i])) > math.Abs(eFunc(extrema[j]))
	})

	uniqueExtrema := make([]float64, 0, len(extrema))
	if len(extrema) > 0 {
		uniqueExtrema = append(uniqueExtrema, extrema[0])
		for i := 1; i < len(extrema); i++ {
			isUnique := true
			for _, v := range uniqueExtrema {
				if math.Abs(v-extrema[i]) < 1e-6 {
					isUnique = false
					break
				}
			}
			if isUnique {
				uniqueExtrema = append(uniqueExtrema, extrema[i])
			}
		}
	}
	extrema = uniqueExtrema

	if len(extrema) < n+2 {
		for len(extrema) < n+2 {
			x := a + float64(len(extrema))*(b-a)/float64(n+1)
			extrema = append(extrema, x)
		}
	}
	finalExtrema := extrema[:n+2]
	sort.Float64s(finalExtrema)
	return finalExtrema, histories
}

// brentMinimize: 戻り値に履歴([]BrentStep)を追加
func brentMinimize(f func(float64) float64, a, b, tol float64, logger func(string, ...interface{}), id int) (float64, []BrentStep) {
	const c = 0.3819660112501051
	x := a + c*(b-a)
	w := x
	v := x
	e := 0.0
	fx := f(x)
	fw := fx
	fv := fx
	d := 0.0

	var history []BrentStep

	logger(" >> [極値#%d] Brent探索開始: 初期区間[%.9f, %.9f]\n", id, a, b)

	iterCount := 0
	for iter := 0; iter < 100; iter++ {
		iterCount = iter + 1
		mid := (a + b) / 2.0
		tol1 := tol*math.Abs(x) + tol
		tol2 := 2.0 * tol1

		if math.Abs(x-mid) <= (tol2 - 0.5*(b-a)) {
			logger(" << [極値#%d] Brent収束: iter=%d, x=%.12f, fx=%.5e\n", id, iter, x, fx)
			return x, history
		}

		method := "Golden"

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
				method = "Golden (SPI reject)"
				if x >= mid {
					e = a - x
				} else {
					e = b - x
				}
				d = c * e
			} else {
				method = "SPI"
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
			d = c * e
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

		// ログ出力と履歴保存
		logger("     [#%d Iter %2d] %-20s x=%.12f, fx=%.5e, 区間=[%.9f, %.9f]\n", id, iterCount, method, x, fx, a, b)
		history = append(history, BrentStep{
			Iter:      iterCount,
			Method:    method,
			X:         x,
			Fx:        fx,
			IntervalW: math.Abs(b - a),
		})
	}
	logger(" << [極値#%d] Brent最大反復到達: x=%.12f, fx=%.5e\n", id, x, fx)
	return x, history
}

// --- ユーティリティ & プロット ---

func initializeNodes(n int, a, b float64) []float64 {
	nodes := make([]float64, n+2)
	for k := 0; k < n+2; k++ {
		nodes[k] = (a+b)/2.0 + (b-a)/2.0*math.Cos(float64(k)*math.Pi/float64(n+1))
	}
	sort.Float64s(nodes)
	return nodes
}

// 収束性のチェック
func checkConvergence(prevMaxErr float64, eFunc func(float64) float64, extrema []float64, tolerance float64) (bool, float64) {
	maxErr := 0.0
	for _, x := range extrema {
		err := math.Abs(eFunc(x))
		if err > maxErr {
			maxErr = err
		}
	}
	if math.IsInf(prevMaxErr, 1) {
		return false, maxErr
	}
	diff := math.Abs(maxErr - prevMaxErr)
	return diff < tolerance, maxErr
}

// outputBrentConvergenceBatchPlot: 手法ごとにマーカー形状を変えてプロット
func outputBrentConvergenceBatchPlot(histories map[int][]BrentStep, remesIter int) error {
	p := plot.New()
	p.Title.Text = fmt.Sprintf("Brent Convergence (Interval Width) - Remes Iter %d\n(Circle=SPI, Triangle=Golden)", remesIter)
	p.X.Label.Text = "Brent Iteration"
	p.Y.Label.Text = "Interval Width (Log Scale)"
	p.Y.Scale = plot.LogScale{}
	p.Y.Tick.Marker = plot.LogTicks{}

	colors := []color.Color{
		plotutil.Color(0), plotutil.Color(1), plotutil.Color(2),
		plotutil.Color(3), plotutil.Color(4), plotutil.Color(5),
	}

	var ids []int
	for id := range histories {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	colorIdx := 0
	for _, id := range ids {
		hist := histories[id]

		// ライン用の全点データ
		ptsAll := make(plotter.XYs, len(hist))

		// SPI法を使った点（丸）
		var ptsSPI plotter.XYs
		// Golden法を使った点（三角）
		var ptsGolden plotter.XYs

		for i, step := range hist {
			x := float64(step.Iter)
			y := step.IntervalW
			if y <= 1e-20 {
				y = 1e-20
			}
			pt := plotter.XY{X: x, Y: y}
			ptsAll[i] = pt

			if step.Method == "SPI" {
				ptsSPI = append(ptsSPI, pt)
			} else {
				ptsGolden = append(ptsGolden, pt)
			}
		}

		c := colors[colorIdx%len(colors)]

		// 1. 線を引く
		line, err := plotter.NewLine(ptsAll)
		if err != nil {
			return err
		}
		line.Color = c
		p.Add(line)

		// 2. SPIの点をプロット (丸)
		if len(ptsSPI) > 0 {
			scatterSPI, err := plotter.NewScatter(ptsSPI)
			if err != nil {
				return err
			}
			scatterSPI.GlyphStyle.Shape = draw.CircleGlyph{}
			scatterSPI.GlyphStyle.Color = c
			scatterSPI.GlyphStyle.Radius = vg.Points(3)
			p.Add(scatterSPI)
		}

		// 3. Goldenの点をプロット (三角)
		if len(ptsGolden) > 0 {
			scatterGolden, err := plotter.NewScatter(ptsGolden)
			if err != nil {
				return err
			}
			scatterGolden.GlyphStyle.Shape = draw.TriangleGlyph{}
			scatterGolden.GlyphStyle.Color = c
			scatterGolden.GlyphStyle.Radius = vg.Points(3)
			p.Add(scatterGolden)
		}

		// 凡例には線だけ追加して色を示す（マーカーを混ぜると煩雑になるため）
		p.Legend.Add(fmt.Sprintf("Ext #%d", id), line)

		colorIdx++
	}

	p.Add(plotter.NewGrid())
	filename := fmt.Sprintf("brent_conv_iter%02d.png", remesIter)
	return p.Save(6*vg.Inch, 4*vg.Inch, filename)
}

// 画像出力用関数
func outputIterationPlots(coeffs []float64, iter int, a, b float64, gridSize int) error {
	ptsTarget := make(plotter.XYs, gridSize+1)
	ptsPoly := make(plotter.XYs, gridSize+1)
	ptsErr := make(plotter.XYs, gridSize+1)
	h := (b - a) / float64(gridSize)
	for i := 0; i <= gridSize; i++ {
		x := a + float64(i)*h
		ptsTarget[i].X = x
		ptsTarget[i].Y = targetFunction(x)
		ptsPoly[i].X = x
		ptsPoly[i].Y = polynomial(coeffs, x, a, b)
		ptsErr[i].X = x
		ptsErr[i].Y = ptsPoly[i].Y - ptsTarget[i].Y
	}

	// 1. 関数と近似式のプロット
	p := plot.New()
	p.Title.Text = fmt.Sprintf("Iter %d Approximation", iter)
	p.X.Label.Text = "x"
	p.Y.Label.Text = "y"
	lineTarget, err := plotter.NewLine(ptsTarget)
	if err != nil {
		return err
	}
	lineTarget.Color = plotutil.Color(0)
	linePoly, err := plotter.NewLine(ptsPoly)
	if err != nil {
		return err
	}
	linePoly.Color = plotutil.Color(1)
	linePoly.LineStyle.Dashes = []vg.Length{vg.Points(2), vg.Points(2)}
	p.Add(lineTarget, linePoly)
	p.Legend.Add("Target", lineTarget)
	p.Legend.Add("Approx", linePoly)
	if err := p.Save(6*vg.Inch, 4*vg.Inch, fmt.Sprintf("iter%02d-func.png", iter)); err != nil {
		return err
	}

	// 2. 誤差関数のプロット（マーカーなし）
	p2 := plot.New()
	p2.Title.Text = fmt.Sprintf("Iter %d Error", iter)
	p2.X.Label.Text = "x"
	p2.Y.Label.Text = "Error"
	lineErr, err := plotter.NewLine(ptsErr)
	if err != nil {
		return err
	}
	lineErr.Color = plotutil.Color(2)
	p2.Add(lineErr)
	p2.Add(plotter.NewGrid())
	if err := p2.Save(6*vg.Inch, 4*vg.Inch, fmt.Sprintf("iter%02d-error.png", iter)); err != nil {
		return err
	}
	return nil
}

// 収束プロット
func outputConvergencePlot(maxErrors []float64) error {
	p := plot.New()
	p.Title.Text = "Convergence History (Max Error)"
	p.X.Label.Text = "Iteration"
	p.Y.Label.Text = "Max Error (Log Scale)"
	p.Y.Scale = plot.LogScale{}
	p.Y.Tick.Marker = plot.LogTicks{}

	pts := make(plotter.XYs, len(maxErrors))
	for i, e := range maxErrors {
		pts[i].X = float64(i + 1)
		pts[i].Y = e
	}
	line, err := plotter.NewLine(pts)
	if err != nil {
		return err
	}
	line.Color = plotutil.Color(3)
	line.Width = vg.Points(2)

	p.Add(line, plotter.NewGrid())
	return p.Save(6*vg.Inch, 4*vg.Inch, "convergence.png")
}

// E(x)誤差ばらつきプロット
func outputSpreadPlot(spreads []float64) error {
	p := plot.New()
	p.Title.Text = "Error Spread History (Max|E| - Min|E|)"
	p.X.Label.Text = "Iteration"
	p.Y.Label.Text = "Spread (Log Scale)"
	p.Y.Scale = plot.LogScale{}
	p.Y.Tick.Marker = plot.LogTicks{}

	pts := make(plotter.XYs, len(spreads))
	for i, s := range spreads {
		pts[i].X = float64(i + 1)
		val := s
		if val <= 0 {
			val = 1e-20
		}
		pts[i].Y = val
	}
	line, err := plotter.NewLine(pts)
	if err != nil {
		return err
	}
	line.Color = plotutil.Color(4)
	line.Width = vg.Points(2)

	p.Add(line, plotter.NewGrid())
	return p.Save(6*vg.Inch, 4*vg.Inch, "convergence_spread.png")
}

// 参照点推移プロット
func outputNodesHistoryPlot(history [][]float64) error {
	p := plot.New()
	p.Title.Text = "Reference Nodes History"
	p.X.Label.Text = "Iteration"
	p.Y.Label.Text = "Node Position"

	numNodes := len(history[0])
	for k := 0; k < numNodes; k++ {
		pts := make(plotter.XYs, len(history))
		for i, nodes := range history {
			pts[i].X = float64(i + 1)
			pts[i].Y = nodes[k]
		}
		line, err := plotter.NewLine(pts)
		if err != nil {
			return err
		}
		line.Color = plotutil.Color(k % 7)
		p.Add(line)
	}

	p.Add(plotter.NewGrid())
	return p.Save(6*vg.Inch, 4*vg.Inch, "nodes_history.png")
}

// 浮動小数点スライスを指定精度でフォーマット
func formatFloatSlice(xs []float64, prec int) string {
	if len(xs) == 0 {
		return "[]"
	}
	f := fmt.Sprintf("%%.%df", prec)
	parts := make([]string, len(xs))
	for i, v := range xs {
		parts[i] = fmt.Sprintf(f, v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// 多項式をLaTeX形式でフォーマット
func formatPolynomialLatex(coeffs []float64, name, variable string, precision int, zeroThresh float64, simplifyUnit bool) string {
	if len(coeffs) == 0 {
		return name + " = 0"
	}
	terms := make([]string, 0, len(coeffs))
	format := fmt.Sprintf("%%.%df", precision)
	for power, c := range coeffs {
		if math.Abs(c) < zeroThresh {
			continue
		}
		sign := "+"
		if c < 0 {
			sign = "-"
		}
		absC := math.Abs(c)
		coefStr := fmt.Sprintf(format, absC)

		if simplifyUnit && math.Abs(absC-1.0) < zeroThresh {
			if power != 0 {
				coefStr = ""
			}
		}

		term := fmt.Sprintf("T_{%d}(%s)", power, variable)

		if len(terms) == 0 {
			if sign == "-" {
				terms = append(terms, "-"+coefStr+term)
			} else {
				terms = append(terms, coefStr+term)
			}
		} else {
			terms = append(terms, sign+" "+coefStr+term)
		}
	}
	return name + " = " + strings.Join(terms, " ")
}
