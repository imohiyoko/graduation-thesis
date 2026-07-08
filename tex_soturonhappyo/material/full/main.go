package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// ---------------------------------------------------------
// 設定
// ---------------------------------------------------------
const (
	OutputDir      = "output"
	ImageName      = "comparison_plot.png"
	WidthImageName = "width_plot.png"
	TexName        = "comparison-slide.tex"

	// LaTeX親ファイルから見た画像パスのプレフィックス
	LatexImgPathPrefix = "material/full/output/" // 実際の画像出力先

	// 目標誤差
	TargetTol = 1e-8
)

// グローバル変数：真の最小値
var TrueMinX float64

// 初期化関数
func init() {
	bestF := f(0.0)
	TrueMinX = 0.0
	for x := 0.0; x <= 5.0; x += 0.00001 {
		if fx := f(x); fx < bestF {
			bestF = fx
			TrueMinX = x
		}
	}
}

// ---------------------------------------------------------
// 目的関数 f(x)
// ---------------------------------------------------------
func f(x float64) float64 {
	return math.Pow(x-2, 2) + 0.5*math.Sin(3*x)
}

// ---------------------------------------------------------
// 黄金分割法の推移記録用構造体
// ---------------------------------------------------------
type GoldenIteration struct {
	Iter int
	A    float64
	B    float64
	C    float64
	D    float64
	FC   float64
	FD   float64
}

// ---------------------------------------------------------
// Brent法の推移記録用構造体
// ---------------------------------------------------------
type BrentIteration struct {
	Iter int
	A    float64
	B    float64
	X    float64
	W    float64
	V    float64
	U    float64
}

// ---------------------------------------------------------
// 1. 黄金分割法 (誤差履歴と幅履歴と推移を返す)
// ---------------------------------------------------------
func runGoldenSection(a, b float64) (plotter.XYs, plotter.XYs, float64, float64, []GoldenIteration) {
	errHistory := make(plotter.XYs, 0)
	widthHistory := make(plotter.XYs, 0)
	iterHistory := make([]GoldenIteration, 0)
	initialWidth := b - a
	gr := (math.Sqrt(5) - 1) / 2
	c := b - gr*(b-a)
	d := a + gr*(b-a)
	fc := f(c)
	fd := f(d)

	for i := 0; ; i++ {
		// 区間幅と推定エラーを計算
		width := b - a
		// エラーは区間幅 / 2 で推定（最小値が区間内どこかにあるなら、最大エラーは幅/2）
		estimatedErr := width / 2.0

		// 現在の状態を記録
		errHistory = append(errHistory, plotter.XY{X: float64(i), Y: estimatedErr})
		widthHistory = append(widthHistory, plotter.XY{X: float64(i), Y: width})
		iterHistory = append(iterHistory, GoldenIteration{
			Iter: i,
			A:    a,
			B:    b,
			C:    c,
			D:    d,
			FC:   fc,
			FD:   fd,
		})

		// 収束条件をチェック：tolに達したら終了
		if estimatedErr < TargetTol {
			break
		}

		// 計算処理
		if fc < fd {
			b = d
			d = c
			fd = fc
			c = b - gr*(b-a)
			fc = f(c)
		} else {
			a = c
			c = d
			fc = fd
			d = a + gr*(b-a)
			fd = f(d)
		}
	}
	finalWidth := b - a
	return errHistory, widthHistory, initialWidth, finalWidth, iterHistory
}

// ---------------------------------------------------------
// 2. Brent法 (誤差履歴と幅履歴と推移を返す)
// ---------------------------------------------------------
func runBrent(ax, bx, cx float64) (plotter.XYs, plotter.XYs, float64, float64, []BrentIteration) {
	errHistory := make(plotter.XYs, 0)
	widthHistory := make(plotter.XYs, 0)
	iterHistory := make([]BrentIteration, 0)

	a := math.Min(ax, cx)
	b := math.Max(ax, cx)
	initialWidth := b - a
	x := bx
	w := x
	v := x
	fx := f(x)
	fw := fx
	fv := fx
	d := 0.0
	e := 0.0
	gr := (math.Sqrt(5) - 1) / 2

	for i := 0; ; i++ {
		xm := (a + b) / 2.0
		tol1 := 1e-14
		tol2 := 2.0 * tol1

		width := b - a
		// エラーは区間幅 / 2 で推定
		estimatedErr := width / 2.0

		// 現在の状態を記録
		errHistory = append(errHistory, plotter.XY{X: float64(i), Y: estimatedErr})
		widthHistory = append(widthHistory, plotter.XY{X: float64(i), Y: width})

		// 収束条件をチェック：tolに達したら終了
		if estimatedErr < TargetTol {
			break
		}

		// 計算処理
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
			if math.Abs(p) >= math.Abs(0.5*q*etemp) || p <= q*(a-x) || p >= q*(b-x) {
				goto GOLDEN
			}
			d = p / q
			u := x + d
			if u-a < tol2 || b-u < tol2 {
				d = tol1
				if x >= xm {
					d = -tol1
				}
			}
			goto EVALUATE
		}
	GOLDEN:
		if x >= xm {
			e = a - x
		} else {
			e = b - x
		}
		d = gr * e
	EVALUATE:
		u := 0.0
		if math.Abs(d) >= tol1 {
			u = x + d
		} else {
			if d > 0 {
				u = x + tol1
			} else {
				u = x - tol1
			}
		}
		fu := f(u)

		// 推移を記録 (uが決定された後)
		iterHistory = append(iterHistory, BrentIteration{
			Iter: i,
			A:    a,
			B:    b,
			X:    x,
			W:    w,
			V:    v,
			U:    u,
		})

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
	finalWidth := b - a
	return errHistory, widthHistory, initialWidth, finalWidth, iterHistory
}

// ---------------------------------------------------------
// 黄金分割法とBrent法の推移テーブルをTXT形式で出力
// ---------------------------------------------------------
func createIterationTable(filename string, goldenIterations []GoldenIteration, brentIterations []BrentIteration) error {
	var txt strings.Builder

	// 黄金分割法の部分
	txt.WriteString("====================================================================\n")
	txt.WriteString("                  黄金分割法の変数推移\n")
	txt.WriteString("====================================================================\n\n")
	txt.WriteString("各イテレーションにおける a, b, c, d, f(c), f(d) の値\n\n")

	// ヘッダー行
	txt.WriteString(fmt.Sprintf("%-6s | %-15s | %-15s | %-15s | %-15s | %-15s | %-15s\n",
		"Iter", "a", "b", "c", "d", "f(c)", "f(d)"))
	txt.WriteString("-------+------------------+------------------+------------------+------------------+------------------+------------------\n")

	// データ行
	for _, iter := range goldenIterations {
		txt.WriteString(fmt.Sprintf("%-6d | %15.10f | %15.10f | %15.10f | %15.10f | %15.10f | %15.10f\n",
			iter.Iter, iter.A, iter.B, iter.C, iter.D, iter.FC, iter.FD))
	}

	txt.WriteString("\n====================================================================\n")
	txt.WriteString(fmt.Sprintf("総イテレーション数: %d\n", len(goldenIterations)))
	txt.WriteString("====================================================================\n\n\n")

	// Brent法の部分
	txt.WriteString("====================================================================\n")
	txt.WriteString("                  Brent法の変数推移\n")
	txt.WriteString("====================================================================\n\n")
	txt.WriteString("各イテレーションにおける a*, b*, x, w, v, u の値\n\n")

	// ヘッダー行
	txt.WriteString(fmt.Sprintf("%-6s | %-15s | %-15s | %-15s | %-15s | %-15s | %-15s\n",
		"Iter", "a*", "b*", "x", "w", "v", "u"))
	txt.WriteString("-------+------------------+------------------+------------------+------------------+------------------+------------------\n")

	// データ行
	for _, iter := range brentIterations {
		txt.WriteString(fmt.Sprintf("%-6d | %15.10f | %15.10f | %15.10f | %15.10f | %15.10f | %15.10f\n",
			iter.Iter, iter.A, iter.B, iter.X, iter.W, iter.V, iter.U))
	}

	txt.WriteString("\n====================================================================\n")
	txt.WriteString(fmt.Sprintf("総イテレーション数: %d\n", len(brentIterations)))
	txt.WriteString("====================================================================\n")

	return os.WriteFile(filename, []byte(txt.String()), 0644)
}

// ---------------------------------------------------------
// グラフ描画 (片対数プロット)
// ---------------------------------------------------------
func plotComparison(filename string, goldenHist, brentHist plotter.XYs) error {
	p := plot.New()
	p.Title.Text = "Convergence Comparison (Log Scale)"
	p.X.Label.Text = "Iteration Count"
	p.Y.Label.Text = "Error |x - x_true|"

	p.Y.Scale = plot.LogScale{}
	p.Y.Tick.Marker = plot.LogTicks{}

	// 黄金分割法 (青線)
	lineG, pointsG, err := plotter.NewLinePoints(goldenHist)
	if err != nil {
		return err
	}
	lineG.Color = color.RGBA{B: 255, A: 255}
	pointsG.Shape = drawRing{}
	pointsG.Color = color.RGBA{B: 255, A: 255}
	p.Add(lineG, pointsG)
	p.Legend.Add("Golden Section", lineG)

	// Brent法 (赤線)
	lineB, pointsB, err := plotter.NewLinePoints(brentHist)
	if err != nil {
		return err
	}
	lineB.Color = color.RGBA{R: 255, A: 255}
	pointsB.Shape = drawRing{}
	pointsB.Color = color.RGBA{R: 255, A: 255}
	p.Add(lineB, pointsB)
	p.Legend.Add("Brent's Method", lineB)

	p.X.Min = 0
	p.Y.Min = 1e-9
	p.Y.Max = 10

	return p.Save(6*vg.Inch, 4*vg.Inch, filename)
}

// ---------------------------------------------------------
// グラフ描画 (区間幅のプロット)
// ---------------------------------------------------------
func plotWidthComparison(filename string, goldenWidth, brentWidth plotter.XYs) error {
	p := plot.New()
	p.Title.Text = "Interval Width Reduction (Log Scale)"
	p.X.Label.Text = "Iteration Count"
	p.Y.Label.Text = "Bracket Width |b - a|"

	p.Y.Scale = plot.LogScale{}
	p.Y.Tick.Marker = plot.LogTicks{}

	// 黄金分割法 (青線)
	lineG, pointsG, err := plotter.NewLinePoints(goldenWidth)
	if err != nil {
		return err
	}
	lineG.Color = color.RGBA{B: 255, A: 255}
	pointsG.Shape = drawRing{}
	pointsG.Color = color.RGBA{B: 255, A: 255}
	p.Add(lineG, pointsG)
	p.Legend.Add("Golden Section", lineG)

	// Brent法 (赤線)
	lineB, pointsB, err := plotter.NewLinePoints(brentWidth)
	if err != nil {
		return err
	}
	lineB.Color = color.RGBA{R: 255, A: 255}
	pointsB.Shape = drawRing{}
	pointsB.Color = color.RGBA{R: 255, A: 255}
	p.Add(lineB, pointsB)
	p.Legend.Add("Brent's Method", lineB)

	p.X.Min = 0
	p.Y.Min = 1e-10
	p.Y.Max = 10

	return p.Save(6*vg.Inch, 4*vg.Inch, filename)
}

// ---------------------------------------------------------
// カスタムマーカー (修正済み)
// ---------------------------------------------------------
type drawRing struct{}

// 修正: DrawGlyphの引数型を最新のgonumに合わせて変更
func (drawRing) DrawGlyph(c *draw.Canvas, sty draw.GlyphStyle, pt vg.Point) {
	// 線スタイルを設定してから描画
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
// LaTeX 生成 (3フレーム: テーブル + 画像1枚 + 考察)
// ---------------------------------------------------------
func createTex(filename string, goldenIter int, goldenTime time.Duration, brentIter int, brentTime time.Duration) error {
	imgNameWidth := LatexImgPathPrefix + WidthImageName

	// PDFと同じ階層にあるTXTファイルへの相対パス
	iterTablePath := "brent_iterations.txt"

	speedupIter := float64(goldenIter) / float64(brentIter)
	speedupTime := goldenTime.Seconds() / brentTime.Seconds()

	content := fmt.Sprintf(`
%% 比較スライド (1/3: テーブル)
\begin{frame}{黄金分割法とBrent法の比較}
  \begin{table}
    \centering
    \caption{収束性能の比較 (誤差 $10^{-8}$ 到達まで)}
    \begin{tabular}{|l|r|r|r|} \hline
      \textbf{手法} & \textbf{イテレーション} & \textbf{計算時間 (ms)} & \textbf{収束次数} \\ \hline \hline
      黄金分割法 & %d回 & %.4f & 1.000 \\
      \textbf{Brent法} & \textbf{%d回} & \textbf{%.4f} & $\approx 1.324$ \\ \hline
      \textbf{Golden/Brent} & \textbf{%.2f×} & \textbf{%.2f×} & - \\ \hline
    \end{tabular}
  \end{table}

  \vspace{1em}

  \begin{itemize}
    \item Brent法は黄金分割法と比べ、\textbf{イテレーション数で約 %.1f倍高速}に収束
    \item さらに、\textbf{計算時間では約 %.2f倍高速}に収束
    \item Brent法の放物線補間が効率的に機能していることを示す
    \item \href{run:code %s}{\textcolor{blue}{両手法の変数推移表を参照}}
  \end{itemize}
\end{frame}

%% 比較スライド (2/3: グラフ)
\begin{frame}{黄金分割法とBrent法の比較：収束履歴}
  \begin{figure}
    \centering
    \includegraphics[width=0.75\textwidth]{%s}
    \caption{区間幅の縮小}
  \end{figure}
\end{frame}

%% 比較スライド (3/3: 考察)
\begin{frame}{収束性能の考察}
  \begin{itemize}
    \item 黄金分割法は対数軸で直線的に減衰（線形収束）
    \item Brent法は放物線補間が効き始めると急激に減衰（超線形収束）
    \vspace{1em}
    \item \textbf{結論}:
    \begin{itemize}
      \item Brent法は黄金分割法と比べ、イテレーション数で約 %.1f倍高速に収束
      \item 計算時間では約 %.2f倍高速に収束
      \item 放物線補間が効率的に機能していることを示す
    \end{itemize}
  \end{itemize}
\end{frame}
`, goldenIter, goldenTime.Seconds()*1000, brentIter, brentTime.Seconds()*1000,
		speedupIter, speedupTime, speedupIter, speedupTime, iterTablePath, imgNameWidth, speedupIter, speedupTime)
	return os.WriteFile(filename, []byte(content), 0644)
}

// ---------------------------------------------------------
// Main
// ---------------------------------------------------------
func main() {
	if _, err := os.Stat(OutputDir); os.IsNotExist(err) {
		_ = os.Mkdir(OutputDir, 0755)
	}

	fmt.Println("1. Running Algorithms...")

	// 黄金分割法の計測
	startGold := time.Now()
	goldenErr, goldenWidth, goldenInitW, goldenFinalW, goldenIterHistory := runGoldenSection(0.0, 5.0)
	goldenTime := time.Since(startGold)

	// Brent法の計測
	startBrent := time.Now()
	brentErr, brentWidth, brentInitW, brentFinalW, brentIterHistory := runBrent(0.0, 1.5, 5.0)
	brentTime := time.Since(startBrent)

	goldenIter := len(goldenErr)
	brentIter := len(brentErr)
	fmt.Printf("   Golden Section: %d iters (%.6f ms)\n", goldenIter, goldenTime.Seconds()*1000)
	fmt.Printf("      Width: %.4f → %.2e (compression: %.2e×)\n", goldenInitW, goldenFinalW, goldenInitW/goldenFinalW)
	fmt.Printf("   Brent Method: %d iters (%.6f ms)\n", brentIter, brentTime.Seconds()*1000)
	fmt.Printf("      Width: %.4f → %.2e (compression: %.2e×)\n", brentInitW, brentFinalW, brentInitW/brentFinalW)
	fmt.Printf("   Speedup: %.2f× faster (time-based)\n", goldenTime.Seconds()/brentTime.Seconds())

	fmt.Println("\n2. Generating Iteration Table (Golden Section + Brent)...")
	// PDFと同じ階層に出力 (../../ で tex_soturonhappyo/ に移動)
	iterTablePath := filepath.Join("..", "..", "brent_iterations.txt")
	if err := createIterationTable(iterTablePath, goldenIterHistory, brentIterHistory); err != nil {
		panic(err)
	}
	fmt.Printf("   ✓ Iteration table: %s\n", iterTablePath)

	fmt.Println("\n3. Generating Width Plot...")
	widthPath := filepath.Join(OutputDir, WidthImageName)
	if err := plotWidthComparison(widthPath, goldenWidth, brentWidth); err != nil {
		panic(err)
	}
	fmt.Printf("   ✓ Width plot: %s\n", widthPath)

	fmt.Println("\n4. Generating Slide TeX...")
	texPath := filepath.Join(OutputDir, TexName)
	if err := createTex(texPath, goldenIter, goldenTime, brentIter, brentTime); err != nil {
		panic(err)
	}

	fmt.Println("\nDone! Check 'output' directory.")
}
