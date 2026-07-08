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
	OutputDir = "output"
	TexName   = "slide_brent.tex"
	// LaTeX親ファイルから見た画像パスのプレフィックス
	LatexImgPathPrefix = "material/brent/output/"
)

// ---------------------------------------------------------
// データ構造
// ---------------------------------------------------------
type Step struct {
	Iter       int
	A, B       float64               // 現在のブラケット [a, b]
	X, W, V    plotter.XY            // 主要な3点 (xが最良, wが次点, vがその次)
	NewPoint   plotter.XY            // 今回の試行点 u
	MethodUsed string                // "Golden" or "SPI"
	Parabola   func(float64) float64 // SPIで使われた放物線関数 (nil許容)
}

// ---------------------------------------------------------
// 目的関数 f(x)
// ---------------------------------------------------------
func f(x float64) float64 {
	return math.Pow(x-2, 2) + 0.5*math.Sin(3*x)
}

// ---------------------------------------------------------
// Brent法の実装
// ---------------------------------------------------------
func brentSearch(ax, bx, cx float64, maxIter int) []Step {
	steps := make([]Step, 0)

	// 初期化
	a := math.Min(ax, cx)
	b := math.Max(ax, cx)
	x := bx
	w := x
	v := x
	fx := f(x)
	fw := fx
	fv := fx

	d := 0.0 // 前回のステップ幅
	e := 0.0 // 前々回のステップ幅

	gr := (math.Sqrt(5) - 1) / 2 // 黄金比

	// Step 0: 初期状態記録
	steps = append(steps, Step{
		Iter: 0,
		A:    a, B: b,
		X:          plotter.XY{X: x, Y: fx},
		W:          plotter.XY{X: w, Y: fw},
		V:          plotter.XY{X: v, Y: fv},
		NewPoint:   plotter.XY{X: math.NaN(), Y: math.NaN()},
		MethodUsed: "Start",
	})

	for iter := 1; iter <= maxIter; iter++ {
		xm := (a + b) / 2.0
		tol1 := 1e-6*math.Abs(x) + 1e-9 // 許容誤差
		tol2 := 2.0 * tol1

		// 収束判定 (ここでは可視化のためループ回数で終了させるが、実用的な判定はこれ)
		if math.Abs(x-xm) <= (tol2 - 0.5*(b-a)) {
			break
		}

		var u float64
		var parabolaFunc func(float64) float64 = nil
		method := "Golden"

		// --- 放物線補間 (SPI) の試行 ---
		if math.Abs(e) > tol1 {
			r := (x - w) * (fx - fv)
			q := (x - v) * (fx - fw)
			p := (x-v)*q - (x-w)*r
			q = 2.0 * (q - r)
			if q > 0.0 {
				p = -p
			}
			q = math.Abs(q)
			etemp := e
			e = d

			// SPI受容条件のチェック
			// 1. ブラケット内にあるか (a < u < b の厳密なチェックより少し緩く)
			// 2. ステップ幅が前回の半分より小さいか (収束が遅すぎないか)
			if math.Abs(p) >= math.Abs(0.5*q*etemp) || p <= q*(a-x) || p >= q*(b-x) {
				// SPI棄却 -> 黄金分割へ
				goto GOLDEN
			}

			// SPI採用
			d = p / q
			u = x + d
			method = "SPI"

			// 可視化用：採用された放物線の関数を作成
			ptX, ptW, ptV := plotter.XY{X: x, Y: fx}, plotter.XY{X: w, Y: fw}, plotter.XY{X: v, Y: fv}
			parabolaFunc = makeParabola(ptX, ptW, ptV)

			// 端点に近すぎる場合の調整
			if u-a < tol2 || b-u < tol2 {
				d = tol1
				if x >= xm {
					d = -tol1
				}
			}
			goto EVALUATE
		}

	GOLDEN:
		// --- 黄金分割探索 ---
		if x >= xm {
			e = a - x
		} else {
			e = b - x
		}
		d = gr * e
		method = "Golden"

	EVALUATE:
		// --- 新しい点 u の評価と変数の更新 ---
		if math.Abs(d) >= tol1 {
			u = x + d
		} else {
			// 最小ステップ幅の保証
			if d > 0 {
				u = x + tol1
			} else {
				u = x - tol1
			}
		}
		fu := f(u)

		// ステップ記録
		steps = append(steps, Step{
			Iter: iter,
			A:    a, B: b,
			X:          plotter.XY{X: x, Y: fx},
			W:          plotter.XY{X: w, Y: fw},
			V:          plotter.XY{X: v, Y: fv},
			NewPoint:   plotter.XY{X: u, Y: fu},
			MethodUsed: method,
			Parabola:   parabolaFunc,
		})

		// ブラケットと主要点の更新
		if fu <= fx {
			if u >= x {
				a = x
			} else {
				b = x
			}
			v, fv = w, fw
			w, fw = x, fx
			x, fx = u, fu
		} else {
			if u < x {
				a = u
			} else {
				b = u
			}
			if fu <= fw || w == x {
				v, fv = w, fw
				w, fw = u, fu
			} else if fu <= fv || v == x || v == w {
				v, fv = u, fu
			}
		}
	}

	return steps
}

// 3点を通る放物線関数を生成するヘルパー
func makeParabola(p1, p2, p3 plotter.XY) func(float64) float64 {
	// 分母が0になるケース(3点が一直線など)はSPIのロジックで弾かれている前提
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
	p.Title.Text = fmt.Sprintf("Brent Iteration %d: [%s]", step.Iter, step.MethodUsed)
	p.X.Label.Text = "X"
	p.Y.Label.Text = "f(X)"
	p.X.Min = 0
	p.X.Max = 5
	p.Y.Min = -1
	p.Y.Max = 10

	// 1. 現在の探索区間 [A, B] をグレーの帯で表示
	polyPts := plotter.XYs{
		{X: step.A, Y: p.Y.Min}, {X: step.A, Y: p.Y.Max},
		{X: step.B, Y: p.Y.Max}, {X: step.B, Y: p.Y.Min},
	}
	poly, err := plotter.NewPolygon(polyPts)
	if err == nil {
		poly.Color = color.RGBA{R: 240, G: 240, B: 240, A: 255}
		poly.LineStyle.Width = 0
		p.Add(poly)
	}

	// 境界線
	lineStyle := draw.LineStyle{
		Color:  color.RGBA{R: 100, G: 100, B: 100, A: 255},
		Dashes: []vg.Length{vg.Points(2), vg.Points(2)},
	}
	drawLine(p, step.A, lineStyle)
	drawLine(p, step.B, lineStyle)

	// 2. 関数の曲線
	funcLine := plotter.NewFunction(f)
	funcLine.Color = color.RGBA{R: 180, G: 180, B: 180, A: 255}
	funcLine.Width = vg.Points(2)
	p.Add(funcLine)

	// 3. 近似放物線の描画（SPIが採用された場合のみ）
	if step.MethodUsed == "SPI" && step.Parabola != nil {
		paraLine := plotter.NewFunction(step.Parabola)
		paraLine.Color = color.RGBA{R: 0, G: 180, B: 0, A: 255} // 緑色
		paraLine.Width = vg.Points(1.5)
		paraLine.LineStyle.Dashes = []vg.Length{vg.Points(4), vg.Points(4)}
		// 描画範囲をブラケット周辺に制限すると見やすいが、今回は全体に描画
		p.Add(paraLine)
	}

	// 4. 主要な3点 (x, w, v) を青丸で表示
	// 重複を除去してプロット
	ptsMap := make(map[float64]plotter.XY)
	ptsMap[step.X.X] = step.X
	ptsMap[step.W.X] = step.W
	ptsMap[step.V.X] = step.V

	pts := make(plotter.XYs, 0, len(ptsMap))
	for _, pt := range ptsMap {
		pts = append(pts, pt)
	}

	if len(pts) > 0 {
		ptsScatter, _ := plotter.NewScatter(pts)
		ptsScatter.GlyphStyle.Shape = drawRing{}
		ptsScatter.GlyphStyle.Color = color.RGBA{R: 0, G: 0, B: 255, A: 255} // 青
		ptsScatter.GlyphStyle.Radius = vg.Points(4)
		p.Add(ptsScatter)
	}

	// 5. 今回の試行点 u を赤丸で表示
	if !math.IsNaN(step.NewPoint.X) {
		newScatter, _ := plotter.NewScatter(plotter.XYs{step.NewPoint})
		newScatter.GlyphStyle.Shape = drawRing{}
		newScatter.GlyphStyle.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255} // 赤
		newScatter.GlyphStyle.Radius = vg.Points(5)
		p.Add(newScatter)
	}

	return p.Save(6*vg.Inch, 4*vg.Inch, filename)
}

// 垂直線を引くヘルパー
func drawLine(p *plot.Plot, x float64, sty draw.LineStyle) {
	line, err := plotter.NewLine(plotter.XYs{{X: x, Y: p.Y.Min}, {X: x, Y: p.Y.Max}})
	if err == nil {
		line.LineStyle = sty
		p.Add(line)
	}
}

type drawRing struct{}

func (drawRing) DrawGlyph(c *draw.Canvas, sty draw.GlyphStyle, pt vg.Point) {
	c.SetLineStyle(draw.LineStyle{Color: sty.Color, Width: vg.Points(1.5)})
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
% Brent法のアニメーション
\begin{frame}{Brent法：ハイブリッド戦略の可視化}
  \begin{columns}
    \begin{column}{0.4\textwidth}
      \begin{itemize}
        \item \textbf{グレー領域}: 解を囲む区間 $[a, b]$
        \item \textcolor{blue}{\textbf{青丸}}: 現在の主要な3点 ($x, w, v$)
        \item \textcolor{red}{\textbf{赤丸}}: 今回の試行点 $u$
        \item \textcolor{green}{\textbf{緑破線}}: 採用された近似放物線
      \end{itemize}
      \vspace{1em}
      \small
      基本は放物線補間を試み、条件が悪い場合（緑線が出ない回）は安全な黄金分割を選択している様子がわかる。
    \end{column}
    \begin{column}{0.6\textwidth}
      \centering
`
	for i := 0; i < numSteps; i++ {
		imgName := fmt.Sprintf("%sbrent_%02d.png", LatexImgPathPrefix, i)
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

	fmt.Println("1. Running Brent's Method...")
	// 初期区間 [0, 5], 初期推測点 1.5 (やや左寄りからスタート)
	steps := brentSearch(0.0, 5.0, 1.5, 15)

	fmt.Printf("2. Generating %d plots...\n", len(steps))
	for _, step := range steps {
		fname := filepath.Join(OutputDir, fmt.Sprintf("brent_%02d.png", step.Iter))
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
