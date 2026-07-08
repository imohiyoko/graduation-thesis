package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"path/filepath"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// ==========================================
// ターゲット関数 & パラメータ設定
// ==========================================

// targetFunction は最小化の対象となる関数です．
// f(x) = -(cos(2x) + x(x-2)(x-6)/3)
func targetFunction(x float64) float64 {
	return -(math.Cos(2.0*x) + x*(x-2.0)*(x-6.0)/3.0)
}

// NewDefaultConfig はデフォルト設定を返します
func NewDefaultConfig() Config {
	return Config{
		// 黄金分割法の探索設定
		SearchMin:      -2.5,
		SearchMax:      2.5,
		GoldenTol:      1e-7,
		GoldenMaxIters: 20, // グラフ生成数削減のため20程度に設定
		GoldenRatio:    0.3819660112501051,

		// グラフ描画設定
		PlotXMin:    -3.0,
		PlotXMax:    3.0,
		PlotYMin:    -4.0,
		PlotYMax:    4.0,
		PlotSamples: 1000,
		OutputDir:   "./", // 出力先ディレクトリ
	}
}

// ==========================================
// データ構造体 & 型定義
// ==========================================

type Config struct {
	SearchMin      float64
	SearchMax      float64
	GoldenTol      float64
	GoldenMaxIters int
	GoldenRatio    float64
	PlotXMin       float64
	PlotXMax       float64
	PlotYMin       float64
	PlotYMax       float64
	PlotSamples    int
	OutputDir      string
}

// VizIterData は各反復ステップの状態を記録する構造体です
type VizIterData struct {
	Iter   int
	A, B   float64
	X1, F1 float64 // Left point
	X2, F2 float64 // Right point
	U, FU  float64 // Best point so far
	Method string  // Algorithm name
}

// VizResult は最適化の最終結果です
type VizResult struct {
	X          float64
	F          float64
	Iterations int
}

// ==========================================
// メインロジック
// ==========================================

func main() {
	cfg := NewDefaultConfig()

	// 出力ディレクトリの作成
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		log.Fatal(err)
	}

	var history []VizIterData

	// コールバック関数の定義
	recorder := func(d VizIterData) {
		history = append(history, d)
		width := d.B - d.A
		fmt.Printf("[#%2d] Range=[%.5f, %.5f] Width=%.5e BestX=%.6f F=%.6f\n",
			d.Iter, d.A, d.B, width, d.U, d.FU)
	}

	fmt.Println("黄金分割法による最適化を開始します...")

	// 最適化実行
	res, err := GoldenSectionMinViz(targetFunction, cfg.SearchMin, cfg.SearchMax,
		cfg.GoldenTol, cfg.GoldenMaxIters, cfg.GoldenRatio, recorder)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n=== 最終結果 ===\n最適解 X = %.10f\n最小値 F = %.10f\n反復数   = %d\n",
		res.X, res.F, res.Iterations)

	// 結果ログの保存
	if err := saveResultLog(res, history, cfg.OutputDir); err != nil {
		log.Printf("ログ保存エラー: %v", err)
	} else {
		fmt.Println("ログファイルを保存しました: golden_result.log")
	}

	// プロット生成
	fmt.Println("\nグラフ画像を生成中...")

	curViewXMin, curViewXMax := cfg.PlotXMin, cfg.PlotXMax
	curViewYMin, curViewYMax := cfg.PlotYMin, cfg.PlotYMax

	for _, data := range history {
		// X軸の範囲更新（ズームイン効果）
		tXMin, tXMax := data.A, data.B
		if tXMin > tXMax {
			tXMin, tXMax = tXMax, tXMin
		}
		curViewXMin, curViewXMax = updateViewRangeX(tXMin, tXMax, curViewXMin, curViewXMax)

		// プロット保存
		if err := saveStepPlot(targetFunction, data, cfg, curViewXMin, curViewXMax, curViewYMin, curViewYMax); err != nil {
			log.Printf("Step %d のプロット保存に失敗: %v", data.Iter, err)
		}
	}
	fmt.Println("ステップごとのグラフ生成完了")

	if err := saveIntervalPlot(history, cfg); err != nil {
		log.Printf("全体推移図の保存に失敗: %v", err)
	} else {
		fmt.Println("全体推移図生成完了")
	}
}

// ==========================================
// プロット関連関数
// ==========================================

func updateViewRangeX(targetMin, targetMax, currentMin, currentMax float64) (float64, float64) {
	targetSpan := targetMax - targetMin
	currentSpan := currentMax - currentMin
	targetCenter := (targetMin + targetMax) / 2.0

	// ターゲットが現在の表示範囲外、または小さすぎる場合にズームイン
	isOut := targetMin < currentMin || targetMax > currentMax
	isTooSmall := false
	if currentSpan > 0 && targetSpan > 1e-8 {
		isTooSmall = targetSpan < (currentSpan / 5.0) // 閾値を調整
	}

	if !isOut && !isTooSmall {
		return currentMin, currentMax
	}

	// 新しい範囲を計算（ターゲットの3倍程度）
	baseSpan := targetSpan * 3.0
	if baseSpan < 0.5 {
		baseSpan = 0.5 // 最小スパン制限
	}

	newMin := targetCenter - baseSpan/2.0
	newMax := targetCenter + baseSpan/2.0

	return newMin, newMax
}

func saveStepPlot(f func(float64) float64, data VizIterData, cfg Config, vXMin, vXMax, vYMin, vYMax float64) error {
	p := plot.New()

	p.Title.Text = fmt.Sprintf("Golden Section Step %02d", data.Iter)
	p.X.Label.Text = "x"
	p.Y.Label.Text = "f(x)"

	// 対象関数の描画
	fn := plotter.NewFunction(f)
	fn.Color = color.RGBA{B: 200, A: 255}
	fn.Width = vg.Points(1.5)
	fn.Samples = cfg.PlotSamples
	fn.XMin = vXMin
	fn.XMax = vXMax
	p.Add(fn)

	// 左側評価点（x1）
	x1Pt := plotter.XYs{{X: data.X1, Y: data.F1}}
	sx1, _ := plotter.NewScatter(x1Pt)
	sx1.GlyphStyle.Radius = vg.Points(4)
	sx1.GlyphStyle.Shape = draw.CircleGlyph{}
	sx1.GlyphStyle.Color = color.RGBA{R: 200, G: 100, B: 0, A: 255}
	p.Add(sx1)
	p.Legend.Add("x1", sx1)

	// 右側評価点（x2）
	x2Pt := plotter.XYs{{X: data.X2, Y: data.F2}}
	sx2, _ := plotter.NewScatter(x2Pt)
	sx2.GlyphStyle.Radius = vg.Points(4)
	sx2.GlyphStyle.Shape = draw.CircleGlyph{}
	sx2.GlyphStyle.Color = color.RGBA{R: 100, G: 150, B: 0, A: 255}
	p.Add(sx2)
	p.Legend.Add("x2", sx2)

	// 最良点
	bestPt := plotter.XYs{{X: data.U, Y: data.FU}}
	sbest, _ := plotter.NewScatter(bestPt)
	sbest.GlyphStyle.Radius = vg.Points(5)
	sbest.GlyphStyle.Shape = draw.RingGlyph{} // 中抜き丸
	sbest.GlyphStyle.Color = color.RGBA{R: 255, A: 255}
	p.Add(sbest)
	p.Legend.Add("best", sbest)

	// 軸範囲
	p.X.Min = vXMin
	p.X.Max = vXMax
	p.Y.Min = vYMin
	p.Y.Max = vYMax

	// 探索区間バーの描画
	yBarPos := vYMin + (vYMax-vYMin)*0.05
	barColor := color.RGBA{B: 255, A: 255}

	intervalLine, _ := plotter.NewLine(plotter.XYs{
		{X: data.A, Y: yBarPos}, {X: data.B, Y: yBarPos},
	})
	intervalLine.Color = barColor
	intervalLine.Width = vg.Points(3)
	p.Add(intervalLine)

	p.Legend.Top = true
	p.Legend.Left = true

	fileName := filepath.Join(cfg.OutputDir, fmt.Sprintf("golden-step-%02d.png", data.Iter))
	return p.Save(8*vg.Inch, 5*vg.Inch, fileName)
}

func saveIntervalPlot(history []VizIterData, cfg Config) error {
	p := plot.New()
	p.Title.Text = "Convergence of Interval [a, b]"
	p.X.Label.Text = "x value"
	p.Y.Label.Text = "Iteration"

	maxIter := float64(len(history))

	for _, d := range history {
		// 区間を表す線
		line, _ := plotter.NewLine(plotter.XYs{
			{X: d.A, Y: float64(d.Iter)}, {X: d.B, Y: float64(d.Iter)},
		})
		line.Color = color.RGBA{G: 150, B: 150, A: 200}
		p.Add(line)

		// 最良点
		dot, _ := plotter.NewScatter(plotter.XYs{{X: d.U, Y: float64(d.Iter)}})
		dot.GlyphStyle.Radius = vg.Points(2)
		dot.GlyphStyle.Color = color.RGBA{R: 200, A: 255}
		p.Add(dot)
	}

	p.Y.Min = maxIter + 1
	p.Y.Max = 0 // 上から下へ時間が進むように
	p.X.Min, p.X.Max = cfg.PlotXMin, cfg.PlotXMax

	fileName := filepath.Join(cfg.OutputDir, "golden-interval.png")
	return p.Save(6*vg.Inch, 6*vg.Inch, fileName)
}

// ==========================================
// 黄金分割法の実装
// ==========================================

// GoldenSectionMinViz は黄金分割法による最小化を行い、各ステップを記録します
func GoldenSectionMinViz(f func(float64) float64, a, b, tol float64, maxIter int,
	ratio float64, callback func(VizIterData)) (*VizResult, error) {

	if a > b {
		a, b = b, a
	}

	// 初期の内分点を計算
	// ratio ~ 0.382 の場合:
	// xLeft = a + 0.382(b-a) -> 左寄り
	// xRight = a + 0.618(b-a) -> 右寄り

	// xLeft (左側), xRight (右側)
	xLeft := a + ratio*(b-a)
	xRight := a + (1.0-ratio)*(b-a)

	// 万が一 ratio > 0.5 だった場合の対策
	if xLeft > xRight {
		xLeft, xRight = xRight, xLeft
	}

	fLeft := f(xLeft)
	fRight := f(xRight)

	for iter := 1; iter <= maxIter; iter++ {
		// 終了条件
		if math.Abs(b-a) < tol {
			best := xLeft
			fBest := fLeft
			if fRight < fLeft {
				best = xRight
				fBest = fRight
			}
			return &VizResult{X: best, F: fBest, Iterations: iter}, nil
		}

		// 現在の最良点を追跡（可視化用）
		best := xLeft
		fBest := fLeft
		if fRight < fLeft {
			best = xRight
			fBest = fRight
		}

		// 記録
		rec := VizIterData{
			Iter:   iter,
			A:      a,
			B:      b,
			X1:     xLeft,
			F1:     fLeft,
			X2:     xRight,
			F2:     fRight,
			U:      best,
			FU:     fBest,
			Method: "Golden Section",
		}
		if callback != nil {
			callback(rec)
		}

		// 区間の更新
		if fLeft < fRight {
			// 最小値は [a, xRight] にある
			// 新しい区間: [a, xRight]
			b = xRight
			xRight = xLeft
			fRight = fLeft

			// 新しい xLeft を計算 (区間縮小比率は 1-ratio = 0.618)
			// 新しい区間幅 L' = (1-ratio) * L
			// xLeft は a から ratio * L' の位置
			xLeft = a + ratio*(b-a)
			fLeft = f(xLeft)

		} else {
			// 最小値は [xLeft, b] にある
			// 新しい区間: [xLeft, b]
			a = xLeft
			xLeft = xRight
			fLeft = fRight

			// 新しい xRight
			xRight = a + (1.0-ratio)*(b-a)
			fRight = f(xRight)
		}
	}

	best := xLeft
	fBest := fLeft
	if fRight < fLeft {
		best = xRight
		fBest = fRight
	}

	return &VizResult{X: best, F: fBest, Iterations: maxIter}, nil
}

// ==========================================
// ログファイル出力
// ==========================================

func saveResultLog(res *VizResult, history []VizIterData, outputDir string) error {
	logPath := filepath.Join(outputDir, "golden_result.log")
	f, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "==========================================")
	fmt.Fprintln(f, "  黄金分割法 最適化ログ")
	fmt.Fprintln(f, "==========================================")
	fmt.Fprintf(f, "最終反復数: %d\n", res.Iterations)
	fmt.Fprintf(f, "最適解 X:   %.10f\n", res.X)
	fmt.Fprintf(f, "最小値 F:   %.10f\n", res.F)
	fmt.Fprintln(f, "------------------------------------------")
	fmt.Fprintln(f, "Step | Range [Min, Max]       | Width        | Best X")
	fmt.Fprintln(f, "------------------------------------------")

	for _, d := range history {
		width := d.B - d.A
		fmt.Fprintf(f, "%4d | [%8.5f, %8.5f] | %12.5e | %.8f\n",
			d.Iter, d.A, d.B, width, d.U)
	}
	fmt.Fprintln(f, "==========================================")

	return nil
}
