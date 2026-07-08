package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"sort"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// ---------------------------------------------------------
// 設定
// ---------------------------------------------------------
const (
	OutputDir = "output"
	TexName   = "slide_spi.tex"
	// LaTeX親ファイルから見た画像パスのプレフィックス
	LatexImgPathPrefix = "material/spi/output/"
)

// ---------------------------------------------------------
// データ構造
// ---------------------------------------------------------
type Step struct {
	Iter     int
	Points   []plotter.XY // 現在の3点
	NewPoint plotter.XY   // 計算された頂点
	Status   string       // "Searching", "Converged", "Diverged", "Stuck"
}

// ---------------------------------------------------------
// 目的関数 f(x)
// ---------------------------------------------------------
func f(x float64) float64 {
	return math.Pow(x-2, 2) + 0.5*math.Sin(3*x)
}

// ---------------------------------------------------------
// 放物線補間 (Successive Parabolic Interpolation)
// ---------------------------------------------------------
func parabolicInterpolation(p1, p2, p3 float64, maxIter int) []Step {
	steps := make([]Step, 0)

	// 点をx座標でソートして管理するスライス
	points := []plotter.XY{
		{X: p1, Y: f(p1)},
		{X: p2, Y: f(p2)},
		{X: p3, Y: f(p3)},
	}
	sortPoints(points)

	// Step 0: 初期状態
	steps = append(steps, Step{
		Iter:     0,
		Points:   clonePoints(points),
		NewPoint: plotter.XY{X: math.NaN(), Y: math.NaN()},
		Status:   "Start",
	})

	for i := 1; i <= maxIter; i++ {
		x1, y1 := points[0].X, points[0].Y
		x2, y2 := points[1].X, points[1].Y
		x3, y3 := points[2].X, points[2].Y

		// 逆放物線補間の公式 (頂点uを求める)
		denom := (x2-x1)*(y2-y3) - (x2-x3)*(y2-y1)

		// 分母が0に近い場合（3点が一直線など） -> スタック
		if math.Abs(denom) < 1e-10 {
			steps = append(steps, Step{
				Iter:     i,
				Points:   clonePoints(points),
				NewPoint: plotter.XY{X: math.NaN(), Y: math.NaN()},
				Status:   "Stuck (Linear)", // 一直線
			})
			break
		}

		u := x2 - ((x2-x1)*(x2-x1)*(y2-y3)-(x2-x3)*(x2-x3)*(y2-y1))/(2*denom)
		fu := f(u)
		newPt := plotter.XY{X: u, Y: fu}

		// 状態判定と記録
		status := "Searching"

		// 1. 発散判定（描画範囲外へ飛んだ場合など）
		if u < -1.0 || u > 6.0 {
			status = "Diverged"
		}

		// 2. スタック判定（既存の点と近すぎる場合）
		if math.Abs(u-x1) < 1e-6 || math.Abs(u-x2) < 1e-6 || math.Abs(u-x3) < 1e-6 {
			status = "Stuck (Loop)"
		}

		steps = append(steps, Step{
			Iter:     i,
			Points:   clonePoints(points),
			NewPoint: newPt,
			Status:   status,
		})

		if status != "Searching" {
			break
		}

		// 次の3点を選ぶ（単純な更新ルール：新しい点を加え、関数値が最も悪い点を捨てる）
		// Brent法のような洗練された更新ではなく、SPIの基本的挙動を示すため
		points = append(points, newPt)

		// Yの値でソートして、最も悪い(Yが大きい)点を捨てる
		sort.Slice(points, func(i, j int) bool {
			return points[i].Y < points[j].Y
		})
		points = points[:3] // ベスト3を残す

		// 次のステップのためにX順に戻しておく（公式適用のため）
		sortPoints(points)
	}

	return steps
}

// ヘルパー関数
func sortPoints(pts []plotter.XY) {
	sort.Slice(pts, func(i, j int) bool {
		return pts[i].X < pts[j].X
	})
}

func clonePoints(pts []plotter.XY) []plotter.XY {
	newPts := make([]plotter.XY, len(pts))
	copy(newPts, pts)
	return newPts
}

// ---------------------------------------------------------
// 補間放物線の関数生成
// ---------------------------------------------------------
// 3点を通る2次関数 L(x) を返す
func makeParabola(p1, p2, p3 plotter.XY) func(float64) float64 {
	return func(x float64) float64 {
		term1 := p1.Y * (x - p2.X) * (x - p3.X) / ((p1.X - p2.X) * (p1.X - p3.X))
		term2 := p2.Y * (x - p1.X) * (x - p3.X) / ((p2.X - p1.X) * (p2.X - p3.X))
		term3 := p3.Y * (x - p1.X) * (x - p2.X) / ((p3.X - p1.X) * (p3.X - p2.X))
		return term1 + term2 + term3
	}
}

// ---------------------------------------------------------
// 描画処理
// ---------------------------------------------------------
func saveStepPlot(filename string, step Step) error {
	p := plot.New()
	p.Title.Text = fmt.Sprintf("SPI Iteration %d: %s", step.Iter, step.Status)
	p.X.Label.Text = "X"
	p.Y.Label.Text = "f(X)"
	p.X.Min = 0
	p.X.Max = 5
	p.Y.Min = -1
	p.Y.Max = 10

	// 1. 関数の曲線
	funcLine := plotter.NewFunction(f)
	funcLine.Color = color.RGBA{R: 200, G: 200, B: 200, A: 255}
	funcLine.Width = vg.Points(2)
	p.Add(funcLine)

	// 2. 近似放物線の描画（3点がある場合のみ）
	if len(step.Points) == 3 {
		parabolaFunc := makeParabola(step.Points[0], step.Points[1], step.Points[2])
		paraLine := plotter.NewFunction(parabolaFunc)
		paraLine.Color = color.RGBA{R: 0, G: 180, B: 0, A: 255} // 緑色
		paraLine.Width = vg.Points(1)
		paraLine.LineStyle.Dashes = []vg.Length{vg.Points(4), vg.Points(4)} // 破線
		p.Add(paraLine)
	}

	// 3. 現在の3点（青丸）
	if len(step.Points) > 0 {
		ptsScatter, _ := plotter.NewScatter(plotter.XYs(step.Points))
		ptsScatter.GlyphStyle.Shape = drawRing{}
		ptsScatter.GlyphStyle.Color = color.RGBA{R: 0, G: 0, B: 255, A: 255}
		ptsScatter.GlyphStyle.Radius = vg.Points(4)
		p.Add(ptsScatter)
	}

	// 4. 計算された新しい頂点（赤丸）
	if !math.IsNaN(step.NewPoint.X) {
		// 画面外の場合は矢印などで示したいが、今回は描画範囲内なら表示
		if step.NewPoint.X >= p.X.Min && step.NewPoint.X <= p.X.Max {
			newScatter, _ := plotter.NewScatter(plotter.XYs{step.NewPoint})
			newScatter.GlyphStyle.Shape = drawRing{}
			newScatter.GlyphStyle.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}
			newScatter.GlyphStyle.Radius = vg.Points(5)
			p.Add(newScatter)
		}
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
// Beamer用TeX生成
// ---------------------------------------------------------
func createTex(filename string, numSteps int) error {
	content := `
% 放物線補間 (SPI) のアニメーション
\begin{frame}{放物線補間の収束プロセス}
  \begin{columns}
    \begin{column}{0.4\textwidth}
      \begin{itemize}
        \item \textcolor{blue}{\textbf{青丸}}: 補間に使用する3点
        \item \textcolor{green}{\textbf{緑破線}}: 3点を通る近似放物線
        \item \textcolor{red}{\textbf{赤丸}}: 放物線の頂点 
      \end{itemize}
      \vspace{1em}
      \small
      頂点付近では極めて高速に収束するが、初期値が悪いと放物線が反転して遠くへ飛んだり、更新が停滞したりするリスクがある。
    \end{column}
    \begin{column}{0.6\textwidth}
      \centering
`
	for i := 0; i < numSteps; i++ {
		imgName := fmt.Sprintf("%sspi_%02d.png", LatexImgPathPrefix, i)
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

	fmt.Println("1. Running Parabolic Interpolation...")
	// 初期点: 少し不安定さを出すために広めにとる、あるいは極小値を挟む形にする
	// 0.0, 1.0, 4.5 -> 左側のカーブに引っ張られて極小値(2.0付近)へ向かうはず
	steps := parabolicInterpolation(0.0, 1.0, 4.5, 10)

	fmt.Printf("2. Generating %d plots...\n", len(steps))
	for _, step := range steps {
		fname := filepath.Join(OutputDir, fmt.Sprintf("spi_%02d.png", step.Iter))
		if err := saveStepPlot(fname, step); err != nil {
			panic(err)
		}
	}

	texParams := filepath.Join(OutputDir, TexName)
	fmt.Printf("3. Generating TeX file at %s...\n", texParams)
	if err := createTex(texParams, len(steps)); err != nil {
		panic(err)
	}

	fmt.Println("Done! Check 'output' directory.")
}
