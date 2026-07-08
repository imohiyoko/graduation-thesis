package main

import (
	"fmt"
	"image/color"
	"math"
	"os"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// ============================================================
// 設定：テイラー多項式の最大次数
// ============================================================
const MaxPolynomialDegree = 25

// ============================================================
// 設定：TeX出力対象のテイラー多項式次数（スライスで複数指定可能）
// ============================================================
var OutputDegrees = []int{1, 3, 5, 7, 9, 11} // テイラー展開用に調整可能

// ============================================================
// 設定：テイラー展開の中心点と範囲
// ============================================================
const (
	CenterPoint = 0.0          // 展開中心点 (Taylor展開の基点)
	RangeStart  = -2 * math.Pi // 範囲の開始
	RangeEnd    = 2 * math.Pi  // 範囲の終了
)

// TargetFunc: 近似対象の関数 (sin関数)
// f(x) = sin(x) on [-π/2, π/2]
func TargetFunc(x float64) float64 {
	return math.Sin(x)
}

func main() {
	fmt.Println("=== sin関数のテイラー展開シミュレーション ===")
	runSimulation(MaxPolynomialDegree)
	fmt.Println("=== シミュレーション完了 ===")
}

// runSimulation: テイラー展開シミュレーションを実行
func runSimulation(maxN int) {
	fmt.Printf("\n--- テイラー展開を開始します（中心点: %f）---\n", CenterPoint)

	gridSteps := 1000
	texLines := generateTexHeader()

	for n := 1; n <= maxN; n++ {
		// 1. テイラー多項式グラフの生成
		plotFile := fmt.Sprintf("sin_taylor_n%02d.png", n)
		if err := plotTaylorApproximation(n, RangeStart, RangeEnd, gridSteps, plotFile); err != nil {
			fmt.Printf("Plot Error for n=%d: %v\n", n, err)
		}

		// 2. 誤差グラフの生成
		errorFile := fmt.Sprintf("sin_taylor_err_n%02d.png", n)
		if err := plotTaylorError(n, RangeStart, RangeEnd, gridSteps, errorFile); err != nil {
			fmt.Printf("Error Plot Error for n=%d: %v\n", n, err)
		} else {
			fmt.Printf("N=%02d: Saved %s and %s\n", n, plotFile, errorFile)
		}

		// TeX用行を追加（OutputDegrees に含まれる次数のみ）
		if shouldOutputToTeX(n) {
			actualDegree := 2*n + 1
			texLines = append(texLines,
				"  \\begin{figure}[H]",
				"    \\begin{minipage}[b]{0.48\\linewidth}",
				"      \\centering",
				fmt.Sprintf("      \\includegraphics[width=1.0\\linewidth]{material/sinTaylor/sin_taylor_n%02d.png}", n),
				fmt.Sprintf("      \\caption{展開結果 (%d次)}", actualDegree),
				"    \\end{minipage}",
				"    \\hfill",
				"    \\begin{minipage}[b]{0.48\\linewidth}",
				"      \\centering",
				fmt.Sprintf("      \\includegraphics[width=1.0\\linewidth]{material/sinTaylor/sin_taylor_err_n%02d.png}", n),
				fmt.Sprintf("      \\caption{展開誤差 (%d次)}", actualDegree),
				"    \\end{minipage}",
				fmt.Sprintf("    \\caption{$f(x) = \\sin(x)$のテイラー展開（%d次多項式）}", actualDegree),
				"  \\end{figure}",
				"",
			)
		}
	}

	texLines = append(texLines, "")

	// TeX ファイル生成
	texFileName := "sin_taylor_result.tex"
	if err := os.WriteFile(texFileName, []byte(joinLines(texLines)), 0644); err != nil {
		fmt.Printf("Error writing %s: %v\n", texFileName, err)
	} else {
		fmt.Printf("%s の生成完了\n", texFileName)
	}
}

// generateTexHeader: TeX ファイルの説明部分を生成
func generateTexHeader() []string {
	return []string{
		"% テイラー展開による近似比較図",
		"",
	}
}

// joinLines: 文字列スライスをニューラインで結合
func joinLines(lines []string) string {
	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}

// plotTaylorApproximation: 真の関数とテイラー多項式をプロット
func plotTaylorApproximation(n int, a, b float64, steps int, filename string) error {
	p := plot.New()
	actualDegree := 2*n + 1
	p.Title.Text = fmt.Sprintf("Taylor Expansion (%d-th order polynomial, Center=%.1f)", actualDegree, CenterPoint)
	p.X.Label.Text = "x"
	p.Y.Label.Text = "y"

	// y軸の範囲を-1.5~1.5に固定
	p.Y.Min = -1.5
	p.Y.Max = 1.5

	// 1. 真の関数の描画
	trueLineData := make(plotter.XYs, steps+1)
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		trueLineData[i] = plotter.XY{X: x, Y: TargetFunc(x)}
	}
	lTrue, _ := plotter.NewLine(trueLineData)
	lTrue.Color = color.RGBA{R: 0, G: 0, B: 0, A: 255} // 黒
	p.Add(lTrue)
	p.Legend.Add("Original f(x)", lTrue)

	// 2. テイラー多項式の描画
	taylorLineData := make(plotter.XYs, steps+1)
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		taylorLineData[i] = plotter.XY{X: x, Y: taylorVal(x, n)}
	}

	lTaylor, _ := plotter.NewLine(taylorLineData)
	lTaylor.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255} // 赤
	lTaylor.Dashes = []vg.Length{vg.Points(2), vg.Points(2)}
	p.Add(lTaylor)
	p.Legend.Add(fmt.Sprintf("Taylor Poly (N=%d)", n), lTaylor)

	// 3. 展開中心点のマーク
	scatterData := plotter.XYs{{X: CenterPoint, Y: TargetFunc(CenterPoint)}}
	scatter, _ := plotter.NewScatter(scatterData)
	scatter.GlyphStyle.Color = color.RGBA{R: 255, G: 0, B: 255, A: 255} // マゼンタ
	scatter.GlyphStyle.Radius = vg.Points(4)
	p.Add(scatter)
	p.Legend.Add("Center", scatter)

	// y軸の範囲を最後に再度設定して確実に固定
	p.Y.Min = -1.5
	p.Y.Max = 1.5

	return p.Save(6*vg.Inch, 4*vg.Inch, filename)
}

// plotTaylorError: 誤差関数 e(x) = Tn(x) - f(x) をプロット
func plotTaylorError(n int, a, b float64, steps int, filename string) error {
	p := plot.New()
	actualDegree := 2*n + 1
	p.Title.Text = fmt.Sprintf("Taylor Expansion Error (%d-th order polynomial)", actualDegree)
	p.X.Label.Text = "x"
	p.Y.Label.Text = "e(x) = Tn(x) - f(x)"

	errorData := make(plotter.XYs, steps+1)

	maxAbsErr := 0.0
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		yApprox := taylorVal(x, n)
		yTrue := TargetFunc(x)
		errVal := yApprox - yTrue
		errorData[i] = plotter.XY{X: x, Y: errVal}

		if math.Abs(errVal) > maxAbsErr {
			maxAbsErr = math.Abs(errVal)
		}
	}

	// 誤差グラフの線
	lError, _ := plotter.NewLine(errorData)
	lError.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	p.Add(lError)

	// ゼロライン（基準線）の追加
	zeroLine, _ := plotter.NewLine(plotter.XYs{{X: a, Y: 0}, {X: b, Y: 0}})
	zeroLine.Color = color.RGBA{R: 150, G: 150, B: 150, A: 150}
	p.Add(zeroLine)

	// 展開中心点でのマーク
	scatterData := plotter.XYs{{X: CenterPoint, Y: 0}}
	scatter, _ := plotter.NewScatter(scatterData)
	scatter.GlyphStyle.Color = color.RGBA{R: 255, G: 0, B: 255, A: 255}
	scatter.GlyphStyle.Radius = vg.Points(4)
	p.Add(scatter)

	// 最大絶対誤差を凡例に表示
	p.Legend.Add(fmt.Sprintf("Max Abs Error: %.2e", maxAbsErr), lError)

	return p.Save(6*vg.Inch, 4*vg.Inch, filename)
}

// taylorVal: sin(x)のテイラー展開多項式の値を計算 (中心 x=0)
// T_n(x) = sum_{k=0}^{n} ((-1)^k * x^(2k+1)) / (2k+1)!
func taylorVal(x float64, n int) float64 {
	sum := 0.0
	for k := 0; k <= n; k++ {
		// 分子: (-1)^k * x^(2k+1)
		xPower := math.Pow(x, float64(2*k+1))
		sign := math.Pow(-1, float64(k))
		numerator := sign * xPower

		// 分母: (2k+1)!
		denominator := float64(factorial(2*k + 1))

		sum += numerator / denominator
	}
	return sum
}

// factorial: 階乗を計算
func factorial(n int) int {
	if n <= 1 {
		return 1
	}
	result := 1
	for i := 2; i <= n; i++ {
		result *= i
	}
	return result
}

// shouldOutputToTeX: 指定された次数がTeX出力対象に含まれているかチェック
func shouldOutputToTeX(degree int) bool {
	for _, d := range OutputDegrees {
		if d == degree {
			return true
		}
	}
	return false
}
