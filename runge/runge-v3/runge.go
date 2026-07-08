package main

import (
	"fmt"
	"image/color"
	"math"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// TargetFunc: 近似対象の関数 (ここではルンゲ関数)
func TargetFunc(x float64) float64 {
	return 1.0 / (1.0 + 25.0*x*x)
}

func main() {
	fmt.Println("ルンゲ現象シミュレーション（チェビシェフ偏差点・極値評価モード）を開始します...")

	// 任意の範囲 [a, b] を指定
	rangeStart := -1.0
	rangeEnd := 1.0

	// 評価・描画に使用するグリッド数（刻み数）を指定
	// 数値を大きくすると誤差評価や極値検出の精度が向上します
	gridSteps := 10000

	// 次数N
	maxN := 25
	for n := 1; n <= maxN; n++ {
		filename := fmt.Sprintf("runge_Chebyshev_n%02d.png", n)
		// gridSteps を引数として渡すように変更
		if err := plotRungeChebyshev(n, rangeStart, rangeEnd, gridSteps, filename); err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Saved: %s\n", filename)
		}
	}
}

// plotRungeChebyshev: チェビシェフノードの補間結果をプロットし，誤差の極値情報を表示
// steps 引数を追加
func plotRungeChebyshev(n int, a, b float64, steps int, filename string) error {
	p := plot.New()

	// 軸の設定
	p.X.Label.Text = "x"
	p.Y.Label.Text = "y"
	p.Y.Min, p.Y.Max = -0.5, 1.5 // チェビシェフ補間は収束が良いので範囲は狭めでOK

	// 1. 真の関数の描画 (黒線)
	// 受け取った steps を使用
	trueLinePts := make(plotter.XYs, 0, steps+1)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := a + (b-a)*t
		trueLinePts = append(trueLinePts, plotter.XY{X: x, Y: TargetFunc(x)})
	}
	lTrue, _ := plotter.NewLine(trueLinePts)
	lTrue.Color = color.RGBA{R: 0, G: 0, B: 0, A: 255} // 黒
	lTrue.Width = vg.Points(1.5)
	p.Add(lTrue)
	p.Legend.Add("Original Function", lTrue)

	// 2. チェビシェフ偏差点による補間 (青点線)
	chebX, chebY := generateChebyshevNodes(n, a, b)
	// 受け取った steps を使用
	chebLinePts := interpolateLine(chebX, chebY, a, b, steps)

	lCheb, _ := plotter.NewLine(chebLinePts)
	lCheb.Color = color.RGBA{R: 0, G: 0, B: 255, A: 255} // 青
	lCheb.Dashes = []vg.Length{vg.Points(2), vg.Points(2)}
	p.Add(lCheb)
	p.Legend.Add(fmt.Sprintf("Chebyshev (N=%d)", n), lCheb)

	// 3. 誤差の極値（極大・極小）を検出し，その絶対値の最大・最小を求める
	// まず全点の誤差を計算
	diffs := make([]float64, len(chebLinePts))
	for i, pt := range chebLinePts {
		diffs[i] = pt.Y - TargetFunc(pt.X)
	}

	// 極値（Local Extrema）の絶対値を収集
	var extremaAbs []float64
	for i := 1; i < len(diffs)-1; i++ {
		prev := diffs[i-1]
		curr := diffs[i]
		next := diffs[i+1]

		isMax := curr > prev && curr > next // 極大
		isMin := curr < prev && curr < next // 極小

		if isMax || isMin {
			extremaAbs = append(extremaAbs, math.Abs(curr))
		}
	}

	maxAbsExt := 0.0
	minAbsExt := 0.0

	if len(extremaAbs) > 0 {
		minAbsExt = math.MaxFloat64 // 初期化
		for _, v := range extremaAbs {
			if v > maxAbsExt {
				maxAbsExt = v
			}
			if v < minAbsExt {
				minAbsExt = v
			}
		}
	} else {
		// 極値が見つからない場合のフォールバック（平坦に近い場合など）
		maxVal := 0.0
		for _, d := range diffs {
			if math.Abs(d) > maxVal {
				maxVal = math.Abs(d)
			}
		}
		maxAbsExt = maxVal
		minAbsExt = maxVal
	}

	// タイトルに誤差情報を追記 (指数表記)
	p.Title.Text = fmt.Sprintf("Chebyshev Interpolation (N=%d)\nMax Abs Ext: %.2e, Min Abs Ext: %.2e", n, maxAbsExt, minAbsExt)

	// 凡例の設定
	p.Legend.Top = true
	p.Legend.Left = false

	return p.Save(6*vg.Inch, 5*vg.Inch, filename)
}

// generateChebyshevNodes: 区間[a,b]でチェビシェフ偏差点を生成
func generateChebyshevNodes(n int, a, b float64) ([]float64, []float64) {
	numPoints := n + 1
	xs := make([]float64, numPoints)
	ys := make([]float64, numPoints)

	for k := 0; k < numPoints; k++ {
		// 1. 標準区間 [-1, 1] でのチェビシェフ根 x_k を計算
		numerator := (2.0*float64(k) + 1.0) * math.Pi
		denominator := 2.0 * float64(numPoints)
		xkStd := math.Cos(numerator / denominator)

		// 2. 一般区間 [a, b] への線形写像
		mid := (a + b) / 2.0
		halfRange := (b - a) / 2.0
		xkMapped := mid + halfRange*xkStd

		xs[k] = xkMapped
		ys[k] = TargetFunc(xkMapped)
	}
	return xs, ys
}

// interpolateLine: ラグランジュ補間により，区間[a,b]全体のグラフ曲線を生成
func interpolateLine(nodesX, nodesY []float64, a, b float64, steps int) plotter.XYs {
	pts := make(plotter.XYs, 0, steps+1)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := a + (b-a)*t
		y := lagrangeVal(x, nodesX, nodesY)
		pts = append(pts, plotter.XY{X: x, Y: y})
	}
	return pts
}

// lagrangeVal: ラグランジュ補間計算
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
