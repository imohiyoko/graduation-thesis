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
var OutputDegrees = []int{4, 7, 15} // 表示する次数（調整可能）

// ============================================================
// 設定：補間範囲
// ============================================================
const (
	RangeStart = -1.0 // 範囲の開始
	RangeEnd   = 1.0  // 範囲の終了
)

// TargetFunc: 近似対象の関数 (ルンゲ関数)
// f(x) = 1 / (1 + 25x^2)
func TargetFunc(x float64) float64 {
	return 1.0 / (1.0 + 25.0*x*x)
}

func main() {
	fmt.Println("=== ルンゲ関数の補間シミュレーション ===")

	// ディレクトリを作成
	os.MkdirAll("equidistant", 0755)
	os.MkdirAll("chebyshev", 0755)

	// 等間隔補間
	runEquidistantSimulation(MaxPolynomialDegree)

	// チェビシェフ補間
	runChebyshevSimulation(MaxPolynomialDegree)

	fmt.Println("=== シミュレーション完了 ===")
}

// ============================================================
// 等間隔補間シミュレーション
// ============================================================
func runEquidistantSimulation(maxN int) {
	fmt.Printf("\n--- 等間隔補間を開始します ---\n")

	gridSteps := 1000
	texLines := []string{
		"% ルンゲ関数の等間隔補間による比較図",
		"",
	}

	outputCount := 0
	for n := 1; n <= maxN; n++ {
		// 1. 補間グラフの生成
		plotFile := fmt.Sprintf("equidistant/runge_n%02d.png", n)
		if err := plotEquidistantInterpolation(n, RangeStart, RangeEnd, gridSteps, plotFile); err != nil {
			fmt.Printf("Plot Error for n=%d: %v\n", n, err)
		}

		// 2. 誤差グラフの生成
		errorFile := fmt.Sprintf("equidistant/runge_err_n%02d.png", n)
		if err := plotEquidistantError(n, RangeStart, RangeEnd, gridSteps, errorFile); err != nil {
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
				label = "fig:runge-equidistant-first"
			} else if outputCount == len(OutputDegrees) {
				label = "fig:runge-equidistant-last"
			} else {
				label = fmt.Sprintf("fig:runge-equidistant-%d", outputCount)
			}

			// 説明文を画像の上に配置するように修正
			texLines = append(texLines,
				"\\begin{figure}[H]",
				"  \\centering",
				fmt.Sprintf("  \\caption{$f(x) = \\frac{1}{1+25x^2}$の等間隔補間（$n=%d$次多項式）}", n),
				fmt.Sprintf("  \\label{%s}", label),
				"  \\vspace{5pt}", // キャプションと画像の間隔調整
				"  \\begin{minipage}[b]{0.48\\linewidth}",
				"    \\centering",
				fmt.Sprintf("    \\includegraphics[width=1.0\\linewidth]{material/runge/equidistant/runge_n%02d.png}", n),
				"  \\end{minipage}",
				"  \\hfill",
				"  \\begin{minipage}[b]{0.48\\linewidth}",
				"    \\centering",
				fmt.Sprintf("    \\includegraphics[width=1.0\\linewidth]{material/runge/equidistant/runge_err_n%02d.png}", n),
				"  \\end{minipage}",
				"\\end{figure}",
				"",
			)
		}
	}

	// TeX ファイル生成
	texFileName := "runge_equidistant_result.tex"
	if err := os.WriteFile(texFileName, []byte(joinLines(texLines)), 0644); err != nil {
		fmt.Printf("Error writing %s: %v\n", texFileName, err)
	} else {
		fmt.Printf("%s の生成完了\n", texFileName)
	}
}

// ============================================================
// チェビシェフ補間シミュレーション
// ============================================================
func runChebyshevSimulation(maxN int) {
	fmt.Printf("\n--- チェビシェフ補間を開始します ---\n")

	gridSteps := 1000
	texLines := []string{
		"% ルンゲ関数のチェビシェフ補間による比較図",
		"",
	}

	outputCount := 0
	for n := 1; n <= maxN; n++ {
		// 1. 補間グラフの生成
		plotFile := fmt.Sprintf("chebyshev/runge_cheby_n%02d.png", n)
		if err := plotChebyshevInterpolation(n, RangeStart, RangeEnd, gridSteps, plotFile); err != nil {
			fmt.Printf("Plot Error for n=%d: %v\n", n, err)
		}

		// 2. 誤差グラフの生成
		errorFile := fmt.Sprintf("chebyshev/runge_cheby_err_n%02d.png", n)
		if err := plotChebyshevError(n, RangeStart, RangeEnd, gridSteps, errorFile); err != nil {
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
				label = "fig:runge-chebyshev-first"
			} else if outputCount == len(OutputDegrees) {
				label = "fig:runge-chebyshev-last"
			} else {
				label = fmt.Sprintf("fig:runge-chebyshev-%d", outputCount)
			}

			// 説明文を画像の上に配置するように修正
			texLines = append(texLines,
				"\\begin{figure}[H]",
				"  \\centering",
				fmt.Sprintf("  \\caption{$f(x) = \\frac{1}{1+25x^2}$のチェビシェフ補間（$n=%d$次多項式）}", n),
				fmt.Sprintf("  \\label{%s}", label),
				"  \\vspace{5pt}", // キャプションと画像の間隔調整
				"  \\begin{minipage}[b]{0.48\\linewidth}",
				"    \\centering",
				fmt.Sprintf("    \\includegraphics[width=1.0\\linewidth]{material/runge/chebyshev/runge_cheby_n%02d.png}", n),
				"  \\end{minipage}",
				"  \\hfill",
				"  \\begin{minipage}[b]{0.48\\linewidth}",
				"    \\centering",
				fmt.Sprintf("    \\includegraphics[width=1.0\\linewidth]{material/runge/chebyshev/runge_cheby_err_n%02d.png}", n),
				"  \\end{minipage}",
				"\\end{figure}",
				"",
			)
		}
	}

	// TeX ファイル生成
	texFileName := "runge_chebyshev_result.tex"
	if err := os.WriteFile(texFileName, []byte(joinLines(texLines)), 0644); err != nil {
		fmt.Printf("Error writing %s: %v\n", texFileName, err)
	} else {
		fmt.Printf("%s の生成完了\n", texFileName)
	}
}

// (中略: generateEquidistantNodes, generateChebyshevNodes, lagrangeVal, プロット用関数等は元のコードと同じ)

// ============================================================
// 等間隔補間の計算
// ============================================================
func generateEquidistantNodes(n int, a, b float64) ([]float64, []float64) {
	xs := make([]float64, n+1)
	ys := make([]float64, n+1)
	for i := 0; i <= n; i++ {
		x := a + (b-a)*float64(i)/float64(n)
		xs[i] = x
		ys[i] = TargetFunc(x)
	}
	return xs, ys
}

func generateChebyshevNodes(n int, a, b float64) ([]float64, []float64) {
	numPoints := n + 1
	xs := make([]float64, numPoints)
	ys := make([]float64, numPoints)
	for k := 0; k < numPoints; k++ {
		numerator := (2.0*float64(k) + 1.0) * math.Pi
		denominator := 2.0 * float64(numPoints)
		xkStd := math.Cos(numerator / denominator)
		mid := (a + b) / 2.0
		halfRange := (b - a) / 2.0
		xkMapped := mid + halfRange*xkStd
		xs[k] = xkMapped
		ys[k] = TargetFunc(xkMapped)
	}
	return xs, ys
}

func lagrangeVal(x float64, xs, ys []float64) float64 {
	n := len(xs)
	sum := 0.0
	for i := 0; i < n; i++ {
		term := ys[i]
		for j := 0; j < n; j++ {
			if i != j {
				term *= (x - xs[j]) / (xs[i] - xs[j])
			}
		}
		sum += term
	}
	return sum
}

func plotEquidistantInterpolation(n int, a, b float64, steps int, filename string) error {
	p := plot.New()
	p.Title.Text = fmt.Sprintf("Equidistant Interpolation (n=%d)", n)
	p.X.Label.Text = "x"
	p.Y.Label.Text = "y"
	p.Y.Min = -0.5
	p.Y.Max = 1.2
	ptsTarget := make(plotter.XYs, steps+1)
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		ptsTarget[i].X = x
		ptsTarget[i].Y = TargetFunc(x)
	}
	lineTarget, _ := plotter.NewLine(ptsTarget)
	lineTarget.Color = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	p.Add(lineTarget)
	xs, ys := generateEquidistantNodes(n, a, b)
	ptsInterp := make(plotter.XYs, steps+1)
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		ptsInterp[i].X = x
		ptsInterp[i].Y = lagrangeVal(x, xs, ys)
	}
	lineInterp, _ := plotter.NewLine(ptsInterp)
	lineInterp.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	lineInterp.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}
	p.Add(lineInterp)
	ptsNodes := make(plotter.XYs, len(xs))
	for i, x := range xs {
		ptsNodes[i].X = x
		ptsNodes[i].Y = ys[i]
	}
	scatter, _ := plotter.NewScatter(ptsNodes)
	scatter.Color = color.RGBA{R: 0, G: 0, B: 255, A: 255}
	p.Add(scatter)
	p.Legend.Add("f(x)", lineTarget)
	p.Legend.Add(fmt.Sprintf("Interpolation (n=%d)", n), lineInterp)
	p.Legend.Add("Nodes", scatter)
	err := p.Save(6*vg.Inch, 4*vg.Inch, filename)
	if err != nil {
		return err
	}
	caption := fmt.Sprintf("f(x) = 1/(1+25x^2) Equidistant (n=%d)", n)
	return addTextToImage(filename, caption)
}

func plotEquidistantError(n int, a, b float64, steps int, filename string) error {
	p := plot.New()
	p.Title.Text = fmt.Sprintf("Equidistant Error (n=%d)", n)
	p.X.Label.Text = "x"
	p.Y.Label.Text = "Error"
	xs, ys := generateEquidistantNodes(n, a, b)
	ptsError := make(plotter.XYs, steps+1)
	maxAbsErr := 0.0
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		yApprox := lagrangeVal(x, xs, ys)
		yTrue := TargetFunc(x)
		errVal := yApprox - yTrue
		ptsError[i].X = x
		ptsError[i].Y = errVal
		if math.Abs(errVal) > maxAbsErr {
			maxAbsErr = math.Abs(errVal)
		}
	}
	lineError, _ := plotter.NewLine(ptsError)
	lineError.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	p.Add(lineError)
	zeroLine, _ := plotter.NewLine(plotter.XYs{{X: a, Y: 0}, {X: b, Y: 0}})
	zeroLine.Color = color.RGBA{R: 150, G: 150, B: 150, A: 150}
	p.Add(zeroLine)
	p.Legend.Add(fmt.Sprintf("Max Abs Error: %.2e", maxAbsErr), lineError)
	err := p.Save(6*vg.Inch, 4*vg.Inch, filename)
	if err != nil {
		return err
	}
	caption := fmt.Sprintf("f(x) = 1/(1+25x^2) Equidistant error (n=%d)", n)
	return addTextToImage(filename, caption)
}

func plotChebyshevInterpolation(n int, a, b float64, steps int, filename string) error {
	p := plot.New()
	p.Title.Text = fmt.Sprintf("Chebyshev Interpolation (n=%d)", n)
	p.X.Label.Text = "x"
	p.Y.Label.Text = "y"
	p.Y.Min = -0.5
	p.Y.Max = 1.2
	ptsTarget := make(plotter.XYs, steps+1)
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		ptsTarget[i].X = x
		ptsTarget[i].Y = TargetFunc(x)
	}
	lineTarget, _ := plotter.NewLine(ptsTarget)
	lineTarget.Color = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	p.Add(lineTarget)
	xs, ys := generateChebyshevNodes(n, a, b)
	ptsInterp := make(plotter.XYs, steps+1)
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		ptsInterp[i].X = x
		ptsInterp[i].Y = lagrangeVal(x, xs, ys)
	}
	lineInterp, _ := plotter.NewLine(ptsInterp)
	lineInterp.Color = color.RGBA{R: 0, G: 0, B: 255, A: 255}
	lineInterp.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}
	p.Add(lineInterp)
	ptsNodes := make(plotter.XYs, len(xs))
	for i, x := range xs {
		ptsNodes[i].X = x
		ptsNodes[i].Y = ys[i]
	}
	scatter, _ := plotter.NewScatter(ptsNodes)
	scatter.Color = color.RGBA{R: 0, G: 128, B: 0, A: 255}
	p.Add(scatter)
	p.Legend.Add("f(x)", lineTarget)
	p.Legend.Add(fmt.Sprintf("Interpolation (n=%d)", n), lineInterp)
	p.Legend.Add("Nodes", scatter)
	err := p.Save(6*vg.Inch, 4*vg.Inch, filename)
	if err != nil {
		return err
	}
	caption := fmt.Sprintf("f(x) = 1/(1+25x^2) Chebyshev (n=%d)", n)
	return addTextToImage(filename, caption)
}

func plotChebyshevError(n int, a, b float64, steps int, filename string) error {
	p := plot.New()
	p.Title.Text = fmt.Sprintf("Chebyshev Error (n=%d)", n)
	p.X.Label.Text = "x"
	p.Y.Label.Text = "Error"
	xs, ys := generateChebyshevNodes(n, a, b)
	ptsError := make(plotter.XYs, steps+1)
	maxAbsErr := 0.0
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		yApprox := lagrangeVal(x, xs, ys)
		yTrue := TargetFunc(x)
		errVal := yApprox - yTrue
		ptsError[i].X = x
		ptsError[i].Y = errVal
		if math.Abs(errVal) > maxAbsErr {
			maxAbsErr = math.Abs(errVal)
		}
	}
	lineError, _ := plotter.NewLine(ptsError)
	lineError.Color = color.RGBA{R: 0, G: 0, B: 255, A: 255}
	p.Add(lineError)
	zeroLine, _ := plotter.NewLine(plotter.XYs{{X: a, Y: 0}, {X: b, Y: 0}})
	zeroLine.Color = color.RGBA{R: 150, G: 150, B: 150, A: 150}
	p.Add(zeroLine)
	p.Legend.Add(fmt.Sprintf("Max Abs Error: %.2e", maxAbsErr), lineError)
	err := p.Save(6*vg.Inch, 4*vg.Inch, filename)
	if err != nil {
		return err
	}
	caption := fmt.Sprintf("f(x) = 1/(1+25x^2) Chebyshev error (n=%d)", n)
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
