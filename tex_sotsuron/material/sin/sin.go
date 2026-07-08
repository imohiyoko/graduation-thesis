package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// ============================================================
// 設定：多項式の最大次数
// ============================================================
const MaxPolynomialDegree = 25

// ============================================================
// 設定：TeX出力対象の多項式次数（スライスで複数指定可能）
// ============================================================
var OutputDegrees = []int{1, 5, 11} // 表示する次数（調整可能）

// ============================================================
// 設定：範囲の統一
// ============================================================
const (
	RangeStart = -math.Pi // 範囲の開始
	RangeEnd   = math.Pi  // 範囲の終了
)

// ============================================================
// 設定：テイラー展開の中心点
// ============================================================
const CenterPoint = 0.0 // 展開中心点

// TargetFunc: 近似対象の関数 (sin関数)
func TargetFunc(x float64) float64 {
	return math.Sin(x)
}

func main() {
	fmt.Println("=== sin関数のテイラー展開と補間の統合シミュレーション ===")

	// ディレクトリを作成
	os.MkdirAll("taylor", 0755)
	os.MkdirAll("interp", 0755)

	// テイラー展開
	runTaylorSimulation(MaxPolynomialDegree)

	// 等間隔補間
	runInterpolationSimulation(MaxPolynomialDegree)

	fmt.Println("=== シミュレーション完了 ===")
}

// ============================================================
// テイラー展開シミュレーション
// ============================================================
func runTaylorSimulation(maxN int) {
	fmt.Printf("\n--- テイラー展開を開始します（中心点: %f）---\n", CenterPoint)

	gridSteps := 1000
	texLines := []string{
		"% sin関数のテイラー展開による近似比較図",
		"",
	}

	outputCount := 0
	for n := 1; n <= maxN; n++ {
		// 1. テイラー多項式グラフの生成
		plotFile := fmt.Sprintf("taylor/sin_taylor_n%02d.png", n)
		if err := plotTaylorApproximation(n, RangeStart, RangeEnd, gridSteps, plotFile); err != nil {
			fmt.Printf("Plot Error for n=%d: %v\n", n, err)
		}

		// 2. 誤差グラフの生成
		errorFile := fmt.Sprintf("taylor/sin_taylor_err_n%02d.png", n)
		if err := plotTaylorError(n, RangeStart, RangeEnd, gridSteps, errorFile); err != nil {
			fmt.Printf("Error Plot Error for n=%d: %v\n", n, err)
		} else {
			fmt.Printf("N=%02d: Saved %s and %s\n", n, plotFile, errorFile)
		}

		// TeX用行を追加（OutputDegrees に含まれる次数のみ）
		if shouldOutputToTeX(n) {
			outputCount++
			actualDegree := 2*n + 1

			// ラベルの決定
			var label string
			if outputCount == 1 {
				label = "fig:sin-taylor-first"
			} else if outputCount == len(OutputDegrees) {
				label = "fig:sin-taylor-last"
			} else {
				label = fmt.Sprintf("fig:sin-taylor-%d", outputCount)
			}

			texLines = append(texLines,
				"\\begin{figure}[H]",
				"  \\centering",
				fmt.Sprintf("  \\caption{$f(x) = \\sin(x)$のテイラー展開（%d次多項式）}", actualDegree),
				fmt.Sprintf("  \\label{%s}", label),
				"  \\vspace{5pt}",
				"  \\begin{minipage}[b]{0.48\\linewidth}",
				"    \\centering",
				fmt.Sprintf("    \\includegraphics[width=1.0\\linewidth]{material/sin/taylor/sin_taylor_n%02d.png}", n),
				"  \\end{minipage}",
				"  \\hfill",
				"  \\begin{minipage}[b]{0.48\\linewidth}",
				"    \\centering",
				fmt.Sprintf("    \\includegraphics[width=1.0\\linewidth]{material/sin/taylor/sin_taylor_err_n%02d.png}", n),
				"  \\end{minipage}",
				"\\end{figure}",
				"",
			)
		}
	}

	// TeX ファイル生成
	texFileName := "sin_taylor_result.tex"
	if err := os.WriteFile(texFileName, []byte(joinLines(texLines)), 0644); err != nil {
		fmt.Printf("Error writing %s: %v\n", texFileName, err)
	} else {
		fmt.Printf("%s の生成完了\n", texFileName)
	}
}

// ============================================================
// 補間シミュレーション
// ============================================================
func runInterpolationSimulation(maxN int) {
	fmt.Printf("\n--- 等間隔補間を開始します ---\n")

	gridSteps := 1000
	texLines := []string{
		"% sin関数の等間隔補間による比較図",
		"",
	}

	outputCount := 0
	for n := 1; n <= maxN; n++ {
		// 1. 補間グラフの生成
		plotFile := fmt.Sprintf("interp/sin_interp_n%02d.png", n)
		if err := plotInterpolation(n, RangeStart, RangeEnd, gridSteps, plotFile); err != nil {
			fmt.Printf("Plot Error for n=%d: %v\n", n, err)
		}

		// 2. 誤差グラフの生成
		errorFile := fmt.Sprintf("interp/sin_interp_err_n%02d.png", n)
		if err := plotInterpolationError(n, RangeStart, RangeEnd, gridSteps, errorFile); err != nil {
			fmt.Printf("Error Plot Error for n=%d: %v\n", n, err)
		} else {
			fmt.Printf("N=%02d: Saved %s and %s\n", n, plotFile, errorFile)
		}

		// TeX用行を追加（OutputDegrees に含まれる次数のみ）
		if shouldOutputToTeX(n) {
			outputCount++

			// ラベルの決定
			var label string
			if outputCount == 1 {
				label = "fig:sin-interp-first"
			} else if outputCount == len(OutputDegrees) {
				label = "fig:sin-interp-last"
			} else {
				label = fmt.Sprintf("fig:sin-interp-%d", outputCount)
			}

			texLines = append(texLines,
				"\\begin{figure}[H]",
				"  \\centering",
				fmt.Sprintf("  \\caption{$f(x) = \\sin(x)$の等間隔補間（$n=%d$次多項式）}", n),
				fmt.Sprintf("  \\label{%s}", label),
				"  \\vspace{5pt}",
				"  \\begin{minipage}[b]{0.48\\linewidth}",
				"    \\centering",
				fmt.Sprintf("    \\includegraphics[width=1.0\\linewidth]{material/sin/interp/sin_interp_n%02d.png}", n),
				"  \\end{minipage}",
				"  \\hfill",
				"  \\begin{minipage}[b]{0.48\\linewidth}",
				"    \\centering",
				fmt.Sprintf("    \\includegraphics[width=1.0\\linewidth]{material/sin/interp/sin_interp_err_n%02d.png}", n),
				"  \\end{minipage}",
				"\\end{figure}",
				"",
			)
		}
	}

	// TeX ファイル生成
	texFileName := "sin_interp_result.tex"
	if err := os.WriteFile(texFileName, []byte(joinLines(texLines)), 0644); err != nil {
		fmt.Printf("Error writing %s: %v\n", texFileName, err)
	} else {
		fmt.Printf("%s の生成完了\n", texFileName)
	}
}

// ============================================================
// テイラー多項式の計算
// ============================================================
func taylorPolynomial(x float64, n int) float64 {
	result := 0.0
	for k := 0; k <= n; k++ {
		// sin(x)のテイラー展開：奇数次のみ
		if k%2 == 0 {
			continue
		}
		term := math.Pow(x-CenterPoint, float64(k)) / factorial(k)
		if (k/2)%2 == 1 {
			term = -term
		}
		result += term
	}
	return result
}

// 階乗計算
func factorial(n int) float64 {
	if n == 0 || n == 1 {
		return 1.0
	}
	result := 1.0
	for i := 2; i <= n; i++ {
		result *= float64(i)
	}
	return result
}

// ============================================================
// 等間隔補間の計算
// ============================================================
func lagrangeInterpolation(x float64, xNodes, yNodes []float64) float64 {
	result := 0.0
	n := len(xNodes)
	for i := 0; i < n; i++ {
		term := yNodes[i]
		for j := 0; j < n; j++ {
			if i != j {
				term *= (x - xNodes[j]) / (xNodes[i] - xNodes[j])
			}
		}
		result += term
	}
	return result
}

func generateEquidistantNodes(n int, a, b float64) ([]float64, []float64) {
	xNodes := make([]float64, n+1)
	yNodes := make([]float64, n+1)
	for i := 0; i <= n; i++ {
		xNodes[i] = a + (b-a)*float64(i)/float64(n)
		yNodes[i] = TargetFunc(xNodes[i])
	}
	return xNodes, yNodes
}

// ============================================================
// プロット関数
// ============================================================
func plotTaylorApproximation(n int, a, b float64, steps int, filename string) error {
	actualDegree := 2*n + 1
	p := plot.New()
	p.Title.Text = fmt.Sprintf("Taylor Approximation (degree %d)", actualDegree)
	p.X.Label.Text = "x"
	p.Y.Label.Text = "y"

	// 目標関数
	ptsTarget := make(plotter.XYs, steps+1)
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		ptsTarget[i].X = x
		ptsTarget[i].Y = TargetFunc(x)
	}
	lineTarget, _ := plotter.NewLine(ptsTarget)
	lineTarget.Color = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	p.Add(lineTarget)

	// テイラー多項式
	ptsApprox := make(plotter.XYs, steps+1)
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		ptsApprox[i].X = x
		ptsApprox[i].Y = taylorPolynomial(x, actualDegree)
	}
	lineApprox, _ := plotter.NewLine(ptsApprox)
	lineApprox.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	lineApprox.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}
	p.Add(lineApprox)

	p.Legend.Add("sin(x)", lineTarget)
	p.Legend.Add(fmt.Sprintf("Taylor (n=%d)", actualDegree), lineApprox)

	err := p.Save(6*vg.Inch, 4*vg.Inch, filename)
	if err != nil {
		return err
	}
	caption := fmt.Sprintf("sin(x) Taylor Expansion (n=%d)", actualDegree)
	return addTextToImage(filename, caption)
}

func plotTaylorError(n int, a, b float64, steps int, filename string) error {
	actualDegree := 2*n + 1
	p := plot.New()
	p.Title.Text = fmt.Sprintf("Taylor Approximation Error (degree %d)", actualDegree)
	p.X.Label.Text = "x"
	p.Y.Label.Text = "Error"

	ptsError := make(plotter.XYs, steps+1)
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		ptsError[i].X = x
		ptsError[i].Y = TargetFunc(x) - taylorPolynomial(x, actualDegree)
	}
	lineError, _ := plotter.NewLine(ptsError)
	lineError.Color = color.RGBA{R: 0, G: 0, B: 255, A: 255}
	p.Add(lineError, plotter.NewGrid())

	err := p.Save(6*vg.Inch, 4*vg.Inch, filename)
	if err != nil {
		return err
	}
	caption := fmt.Sprintf("sin(x) Taylor Error (n=%d)", actualDegree)
	return addTextToImage(filename, caption)
}

func plotInterpolation(n int, a, b float64, steps int, filename string) error {
	p := plot.New()
	p.Title.Text = fmt.Sprintf("Lagrange Interpolation (n=%d)", n)
	p.X.Label.Text = "x"
	p.Y.Label.Text = "y"

	// 目標関数
	ptsTarget := make(plotter.XYs, steps+1)
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		ptsTarget[i].X = x
		ptsTarget[i].Y = TargetFunc(x)
	}
	lineTarget, _ := plotter.NewLine(ptsTarget)
	lineTarget.Color = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	p.Add(lineTarget)

	// 補間多項式
	xNodes, yNodes := generateEquidistantNodes(n, a, b)
	ptsInterp := make(plotter.XYs, steps+1)
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		ptsInterp[i].X = x
		ptsInterp[i].Y = lagrangeInterpolation(x, xNodes, yNodes)
	}
	lineInterp, _ := plotter.NewLine(ptsInterp)
	lineInterp.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	lineInterp.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}
	p.Add(lineInterp)

	// 節点
	ptsNodes := make(plotter.XYs, len(xNodes))
	for i, x := range xNodes {
		ptsNodes[i].X = x
		ptsNodes[i].Y = yNodes[i]
	}
	scatter, _ := plotter.NewScatter(ptsNodes)
	scatter.Color = color.RGBA{R: 0, G: 0, B: 255, A: 255}
	p.Add(scatter)

	p.Legend.Add("sin(x)", lineTarget)
	p.Legend.Add(fmt.Sprintf("Interpolation (n=%d)", n), lineInterp)
	p.Legend.Add("Nodes", scatter)

	err := p.Save(6*vg.Inch, 4*vg.Inch, filename)
	if err != nil {
		return err
	}
	caption := fmt.Sprintf("sin(x) Lagrange Interpolation (n=%d)", n)
	return addTextToImage(filename, caption)
}

func plotInterpolationError(n int, a, b float64, steps int, filename string) error {
	p := plot.New()
	p.Title.Text = fmt.Sprintf("Interpolation Error (n=%d)", n)
	p.X.Label.Text = "x"
	p.Y.Label.Text = "Error"

	xNodes, yNodes := generateEquidistantNodes(n, a, b)
	ptsError := make(plotter.XYs, steps+1)
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		ptsError[i].X = x
		ptsError[i].Y = TargetFunc(x) - lagrangeInterpolation(x, xNodes, yNodes)
	}
	lineError, _ := plotter.NewLine(ptsError)
	lineError.Color = color.RGBA{R: 0, G: 0, B: 255, A: 255}
	p.Add(lineError, plotter.NewGrid())

	err := p.Save(6*vg.Inch, 4*vg.Inch, filename)
	if err != nil {
		return err
	}
	caption := fmt.Sprintf("sin(x) Lagrange Error (n=%d)", n)
	return addTextToImage(filename, caption)
}

// ============================================================
// ヘルパー関数
// ============================================================
func shouldOutputToTeX(n int) bool {
	for _, deg := range OutputDegrees {
		if n == deg {
			return true
		}
	}
	return false
}

func joinLines(lines []string) string {
	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}

// addTextToImage adds text to the top of a PNG image
func addTextToImage(filename, text string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)

	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(color.White),
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: fixed.Int26_6(10 * 64), Y: fixed.Int26_6(20 * 64)},
	}

	d.DrawString(text)

	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return png.Encode(outFile, dst)
}
