package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// ---------------------------------------------------------
// 設定
// ---------------------------------------------------------
const (
	// Goプログラムから見た出力先
	OutputDir = "output"
	TexName   = "slide-animation.tex"

	// LaTeX親ファイルから見た画像へのパス (ここを修正)
	// プロジェクトルートから実行する場合の相対パス
	LatexImgPathPrefix = "material/golden/output/"
)

// ---------------------------------------------------------
// データ構造
// ---------------------------------------------------------
type Step struct {
	Iter    int
	A, B    float64
	Current plotter.XY
	History plotter.XYs
}

// ---------------------------------------------------------
// 目的関数 f(x)
// ---------------------------------------------------------
func f(x float64) float64 {
	return math.Pow(x-2, 2) + 0.5*math.Sin(3*x)
}

// ---------------------------------------------------------
// 黄金分割法の実装
// ---------------------------------------------------------
func goldenSectionSearch(a, b float64, iter int) []Step {
	steps := make([]Step, 0)
	history := make(plotter.XYs, 0)

	gr := (math.Sqrt(5) - 1) / 2

	c := b - gr*(b-a)
	d := a + gr*(b-a)

	ptC := plotter.XY{X: c, Y: f(c)}
	ptD := plotter.XY{X: d, Y: f(d)}

	history = append(history, ptC, ptD)

	steps = append(steps, Step{
		Iter:    0,
		A:       a,
		B:       b,
		Current: plotter.XY{X: math.NaN(), Y: math.NaN()},
		History: append(plotter.XYs{}, history...),
	})

	for i := 1; i <= iter; i++ {
		var currentPoint plotter.XY

		if f(c) < f(d) {
			b = d
			d = c
			c = b - gr*(b-a)
			currentPoint = plotter.XY{X: c, Y: f(c)}
		} else {
			a = c
			c = d
			d = a + gr*(b-a)
			currentPoint = plotter.XY{X: d, Y: f(d)}
		}

		currentHistory := make(plotter.XYs, len(history))
		copy(currentHistory, history)

		steps = append(steps, Step{
			Iter:    i,
			A:       a,
			B:       b,
			Current: currentPoint,
			History: currentHistory,
		})

		history = append(history, currentPoint)
	}
	return steps
}

// ---------------------------------------------------------
// グラフ描画と保存
// ---------------------------------------------------------
func saveStepPlot(filename string, step Step) error {
	p := plot.New()
	p.Title.Text = fmt.Sprintf("Iteration %d", step.Iter)
	p.X.Label.Text = "X"
	p.Y.Label.Text = "f(X)"

	p.X.Min = 0
	p.X.Max = 5
	p.Y.Min = -1
	p.Y.Max = 10

	// 1. グレーの探索区間
	polyPts := plotter.XYs{
		{X: step.A, Y: p.Y.Min},
		{X: step.A, Y: p.Y.Max},
		{X: step.B, Y: p.Y.Max},
		{X: step.B, Y: p.Y.Min},
		{X: step.A, Y: p.Y.Min},
	}
	poly, err := plotter.NewPolygon(polyPts)
	if err == nil {
		poly.Color = color.RGBA{R: 240, G: 240, B: 240, A: 255}
		poly.LineStyle.Width = 0
		p.Add(poly)
	}

	// 2. 境界線
	lineA_Pts := plotter.XYs{{X: step.A, Y: p.Y.Min}, {X: step.A, Y: p.Y.Max}}
	if lineA, err := plotter.NewLine(lineA_Pts); err == nil {
		lineA.Color = color.RGBA{R: 100, G: 100, B: 100, A: 255}
		lineA.LineStyle.Dashes = []vg.Length{vg.Points(2), vg.Points(2)}
		p.Add(lineA)
	}

	lineB_Pts := plotter.XYs{{X: step.B, Y: p.Y.Min}, {X: step.B, Y: p.Y.Max}}
	if lineB, err := plotter.NewLine(lineB_Pts); err == nil {
		lineB.Color = color.RGBA{R: 100, G: 100, B: 100, A: 255}
		lineB.LineStyle.Dashes = []vg.Length{vg.Points(2), vg.Points(2)}
		p.Add(lineB)
	}

	// 3. 関数
	funcLine := plotter.NewFunction(f)
	funcLine.Color = color.RGBA{R: 180, G: 180, B: 180, A: 255}
	funcLine.Width = vg.Points(2)
	p.Add(funcLine)

	// 4. 過去の点 (範囲内のみ)
	visibleHistory := make(plotter.XYs, 0)
	for _, pt := range step.History {
		if pt.X >= step.A && pt.X <= step.B {
			visibleHistory = append(visibleHistory, pt)
		}
	}

	if len(visibleHistory) > 0 {
		pastScatter, _ := plotter.NewScatter(visibleHistory)
		pastScatter.GlyphStyle.Shape = drawRing{}
		pastScatter.GlyphStyle.Color = color.RGBA{R: 0, G: 0, B: 255, A: 100}
		pastScatter.GlyphStyle.Radius = vg.Points(3)
		p.Add(pastScatter)
	}

	// 5. 今回の点
	if !math.IsNaN(step.Current.X) {
		currScatter, _ := plotter.NewScatter(plotter.XYs{step.Current})
		currScatter.GlyphStyle.Shape = drawRing{}
		currScatter.GlyphStyle.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}
		currScatter.GlyphStyle.Radius = vg.Points(5)
		p.Add(currScatter)
	}

	return p.Save(6*vg.Inch, 4*vg.Inch, filename)
}

type drawRing struct{}

func (drawRing) DrawGlyph(c *draw.Canvas, sty draw.GlyphStyle, pt vg.Point) {
	c.SetLineStyle(draw.LineStyle{Color: sty.Color, Width: vg.Points(1)})
	c.Stroke(path(pt, sty.Radius))
}
func path(pt vg.Point, rad vg.Length) vg.Path {
	var p vg.Path
	p.Move(pt.Add(vg.Point{X: rad, Y: 0}))
	p.Arc(pt, rad, 0, 2*math.Pi)
	p.Close()
	return p
}

// ---------------------------------------------------------
// Beamerアニメーション用 TeX生成 (パス修正版)
// ---------------------------------------------------------
func createAnimationTex(filename string, numSteps int) error {
	// 注意: 親ファイルから読み込まれる前提なのでプリアンブル(\documentclass等)は削除し
	// frame環境のみを出力するように変更するのが一般的ですが、
	// ここでは要望通りstandaloneでも使える形にしつつ、パスを通します。
	// もし親ファイルで \input するなら、親ファイルのプリアンブルと競合しないよう
	// \begin{frame} ... \end{frame} だけを出力するモードを作るのがベストです。
	// 今回は「親ファイルから参照できるパスを書く」ことに注力します。

	content := `
% このファイルは自動生成されました。
% 親ファイル(tex_soturonhappyoルート)から \input して使用することを想定しています。
\begin{frame}{黄金分割法のアルゴリズム}
  \begin{columns}
    \begin{column}{0.4\textwidth}
      \begin{itemize}
        \item \textbf{グレー領域}: 現在の探索区間
        \item \textcolor{red}{\textbf{赤丸}}: 今回の評価点
        \item \textcolor{blue}{\textbf{青丸}}: 既存の評価点
      \end{itemize}
      \vspace{1em}
      範囲が狭まるにつれ、範囲外の点は探索対象から除外（消去）されます。
    \end{column}
    \begin{column}{0.6\textwidth}
      \centering
`
	for i := 0; i < numSteps; i++ {
		// ここで LatexImgPathPrefix を付与する
		imgName := fmt.Sprintf("%sgolden_%02d.png", LatexImgPathPrefix, i)
		line := fmt.Sprintf("      \\only<%d>{\\includegraphics[width=\\textwidth]{%s}}\n", i+1, imgName)
		content += line
	}

	content += `    \end{column}
  \end{columns}
\end{frame}
`
	return os.WriteFile(filename, []byte(content), 0644)
}

// ---------------------------------------------------------
// Main
// ---------------------------------------------------------
func main() {
	if _, err := os.Stat(OutputDir); os.IsNotExist(err) {
		_ = os.Mkdir(OutputDir, 0755)
	}

	fmt.Println("1. Running Golden Section Search...")
	steps := goldenSectionSearch(0.0, 5.0, 15)

	fmt.Printf("2. Generating %d plots...\n", len(steps))
	for _, step := range steps {
		fname := filepath.Join(OutputDir, fmt.Sprintf("golden_%02d.png", step.Iter))
		if err := saveStepPlot(fname, step); err != nil {
			panic(err)
		}
	}

	texParams := filepath.Join(OutputDir, TexName)
	fmt.Printf("3. Generating TeX file at %s...\n", texParams)
	if err := createAnimationTex(texParams, len(steps)); err != nil {
		panic(err)
	}

	fmt.Println("Done! Check 'output' directory.")
}
