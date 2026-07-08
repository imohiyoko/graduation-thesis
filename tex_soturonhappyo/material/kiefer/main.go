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
)

// ---------------------------------------------------------
// 設定
// ---------------------------------------------------------
const (
	// Goプログラムから見た出力先
	OutputDir = "output"
	TexName   = "frames.tex"

	// LaTeX親ファイルから見た画像へのパス
	LatexImgPathPrefix = "material/kiefer/output/"
)

// ---------------------------------------------------------
// 目的関数 f(x)
// ---------------------------------------------------------
func f(x float64) float64 {
	return math.Pow(x-2, 2) + 0.5*math.Sin(3*x)
}

// ---------------------------------------------------------
// フィボナッチ数列の生成
// ---------------------------------------------------------
func fibonacci(n int) []int {
	fib := make([]int, n)
	if n > 0 {
		fib[0] = 1
	}
	if n > 1 {
		fib[1] = 1
	}
	for i := 2; i < n; i++ {
		fib[i] = fib[i-1] + fib[i-2]
	}
	return fib
}

// ---------------------------------------------------------
// フィボナッチ探索のシミュレーション
// ---------------------------------------------------------
func fibonacci_search(a, b float64, iters int) []map[string]interface{} {
	steps := make([]map[string]interface{}, 0)

	fib := fibonacci(iters + 2)
	fn := float64(fib[iters])
	fn1 := float64(fib[iters+1])

	lambda := a + (fn/fn1)*(b-a)
	mu := a + b - lambda

	eval_lambda := f(lambda)
	eval_mu := f(mu)

	history_points := []plotter.XY{}

	steps = append(steps, map[string]interface{}{
		"iteration":       0,
		"a":               a,
		"b":               b,
		"lambda":          lambda,
		"mu":              mu,
		"eval_lambda":     eval_lambda,
		"eval_mu":         eval_mu,
		"history":         []plotter.XY{},
		"next_type":       "",
		"iteration_total": iters,
	})

	for i := 1; i < iters; i++ {
		if eval_lambda < eval_mu {
			b = mu
			mu = lambda
			eval_mu = eval_lambda

			fn_i := float64(fib[iters-i])
			fn1_i := float64(fib[iters-i+1])
			lambda = a + (fn_i/fn1_i)*(b-a)
			eval_lambda = f(lambda)
		} else {
			a = lambda
			lambda = mu
			eval_lambda = eval_mu

			_ = float64(fib[iters-i])
			_ = float64(fib[iters-i+1])
			mu = a + b - lambda
			eval_mu = f(mu)
		}

		history_points = append(history_points, plotter.XY{X: lambda, Y: eval_lambda})

		steps = append(steps, map[string]interface{}{
			"iteration":       i,
			"a":               a,
			"b":               b,
			"lambda":          lambda,
			"mu":              mu,
			"eval_lambda":     eval_lambda,
			"eval_mu":         eval_mu,
			"history":         history_points,
			"iteration_total": iters,
		})
	}

	return steps
}

// ---------------------------------------------------------
// 黄金比を用いた探索
// ---------------------------------------------------------
func golden_ratio_search(a, b float64, iters int) []map[string]interface{} {
	steps := make([]map[string]interface{}, 0)

	gr := (math.Sqrt(5) - 1) / 2

	lambda := a + (1-gr)*(b-a)
	mu := a + gr*(b-a)

	eval_lambda := f(lambda)
	eval_mu := f(mu)

	history_points := []plotter.XY{}

	steps = append(steps, map[string]interface{}{
		"iteration":   0,
		"a":           a,
		"b":           b,
		"lambda":      lambda,
		"mu":          mu,
		"eval_lambda": eval_lambda,
		"eval_mu":     eval_mu,
		"history":     []plotter.XY{},
	})

	for i := 1; i < iters; i++ {
		if eval_lambda < eval_mu {
			b = mu
			mu = lambda
			eval_mu = eval_lambda

			lambda = a + (1-gr)*(b-a)
			eval_lambda = f(lambda)
		} else {
			a = lambda
			lambda = mu
			eval_lambda = eval_mu

			mu = a + gr*(b-a)
			eval_mu = f(mu)
		}

		history_points = append(history_points, plotter.XY{X: lambda, Y: eval_lambda})

		steps = append(steps, map[string]interface{}{
			"iteration":   i,
			"a":           a,
			"b":           b,
			"lambda":      lambda,
			"mu":          mu,
			"eval_lambda": eval_lambda,
			"eval_mu":     eval_mu,
			"history":     history_points,
		})
	}

	return steps
}

// ---------------------------------------------------------
// グラフ描画
// ---------------------------------------------------------
func drawPlot(title string, search_type string, step map[string]interface{}, filename string) error {
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = "X"
	p.Y.Label.Text = "f(X)"

	p.X.Min = 0
	p.X.Max = 5
	p.Y.Min = -1
	p.Y.Max = 10

	// グレーの探索区間
	polyPts := plotter.XYs{
		{X: step["a"].(float64), Y: p.Y.Min},
		{X: step["a"].(float64), Y: p.Y.Max},
		{X: step["b"].(float64), Y: p.Y.Max},
		{X: step["b"].(float64), Y: p.Y.Min},
		{X: step["a"].(float64), Y: p.Y.Min},
	}
	poly, err := plotter.NewPolygon(polyPts)
	if err == nil {
		poly.Color = color.RGBA{R: 240, G: 240, B: 240, A: 255}
		poly.LineStyle.Width = 0
		p.Add(poly)
	}

	// 関数曲線
	funcLine := plotter.NewFunction(f)
	funcLine.Color = color.RGBA{R: 180, G: 180, B: 180, A: 255}
	funcLine.Width = vg.Points(2)
	p.Add(funcLine)

	// 過去の評価点（青）
	historyPts := step["history"].([]plotter.XY)
	if len(historyPts) > 0 {
		s, err := plotter.NewScatter(plotter.XYs(historyPts))
		if err == nil {
			s.GlyphStyle.Color = color.RGBA{R: 50, G: 100, B: 200, A: 200}
			s.GlyphStyle.Radius = vg.Points(4)
			p.Add(s)
		}
	}

	// 現在の評価点（赤丸と青丸）
	currentPts := plotter.XYs{
		{X: step["lambda"].(float64), Y: step["eval_lambda"].(float64)},
		{X: step["mu"].(float64), Y: step["eval_mu"].(float64)},
	}
	s, err := plotter.NewScatter(currentPts)
	if err == nil {
		s.GlyphStyle.Color = color.RGBA{R: 200, G: 50, B: 50, A: 255}
		s.GlyphStyle.Radius = vg.Points(5)
		p.Add(s)
	}

	// 保存
	if err := os.MkdirAll(OutputDir, 0755); err != nil {
		return err
	}

	return p.Save(10*vg.Centimeter, 8*vg.Centimeter, filepath.Join(OutputDir, filename))
}

// ---------------------------------------------------------
// メイン
// ---------------------------------------------------------
func main() {
	os.MkdirAll(OutputDir, 0755)

	// フレーム1: フィボナッチ法（観測回数が固定）
	fib_steps := fibonacci_search(0, 5, 5)
	drawPlot(
		"フィボナッチ法（観測回数固定時の最適探索）",
		"fibonacci",
		fib_steps[0],
		"fibonacci_fixed_obs.png",
	)

	// フレーム2: 黄金比を用いた方法
	golden_steps := golden_ratio_search(0, 5, 5)
	drawPlot(
		"黄金比を用いたアルゴリズム",
		"golden",
		golden_steps[0],
		"golden_ratio_method.png",
	)

	// フレーム3: 黄金分割法（観測回数を事前に決めない）
	drawPlot(
		"黄金分割法（無限探索、観測回数未決定）",
		"golden",
		golden_steps[min(4, len(golden_steps)-1)],
		"golden_section_no_fixed_obs.png",
	)

	// LaTeXコードの生成
	generateTeX()

	fmt.Printf("✓ 画像生成完了: %s\n", OutputDir)
	fmt.Printf("✓ LaTeX統合ファイル: %s/%s\n", OutputDir, TexName)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ---------------------------------------------------------
// LaTeXコード生成
// ---------------------------------------------------------
func generateTeX() error {
	content := `% このファイルは自動生成されました
% 親ファイル(tex_soturonhappyoルート)から \input して使用することを想定しています

% フレーム1: フィボナッチ法（補足1）
\begin{frame}
  \frametitle{フィボナッチ法による最適探索}
  \begin{columns}
    \begin{column}{0.45\textwidth}
      \small
      \textbf{補足：} Kiefer氏は観測回数が固定された場合、フィボナッチ数を用いた探索法を提示しました。
      
      \vspace{0.5em}
      \begin{itemize}
        \item 観測回数が\textbf{事前に決定}されている
        \item フィボナッチ数により最悪ケースを最適化
        \item 関数値が小さい側に拡大
        \item 赤点：新規評価点
      \end{itemize}
      
      \vspace{0.5em}
      \textbf{目的関数:} $f(x) = (x-2)^2 + 0.5\sin(3x)$
    \end{column}
    \begin{column}{0.55\textwidth}
      \centering
      \includegraphics[width=\textwidth]{material/kiefer/output/fibonacci_fixed_obs.png}
    \end{column}
  \end{columns}
\end{frame}

% フレーム2: 黄金比を用いた方法（補足2）
\begin{frame}
  \frametitle{黄金比を用いたアルゴリズム}
  \begin{columns}
    \begin{column}{0.45\textwidth}
      \small
      \textbf{補足：} これとは別に、黄金比を用いて常に同じアルゴリズムで区間を縮小する手順が提示されました。
      
      \vspace{0.5em}
      \begin{itemize}
        \item 黄金比 $\phi = \frac{\sqrt{5}-1}{2} \approx 0.618$
        \item \textbf{同じアルゴリズム}を繰り返す
        \item 観測回数は事前に決めない
        \item 関数が単峰性を保つ限り有効
      \end{itemize}
      
      \vspace{0.5em}
      これが一般に\textbf{黄金分割法}と呼ばれます。
    \end{column}
    \begin{column}{0.55\textwidth}
      \centering
      \includegraphics[width=\textwidth]{material/kiefer/output/golden_ratio_method.png}
    \end{column}
  \end{columns}
\end{frame}

% フレーム3: 黄金分割法の実用性（補足3）
\begin{frame}
  \frametitle{黄金分割法の実用性}
  \begin{columns}
    \begin{column}{0.45\textwidth}
      \small
      \textbf{補足：} 黄金分割法は厳密には最適な手順ではありませんが、観測回数を事前に決めない場合に特に有用です。
      
      \vspace{0.5em}
      \begin{itemize}
        \item 各段階で区間は最悪でも $0.618$ 倍に縮小する
        \item 実装が簡単で安定している
        \item 収束性が良好
        \item 観測回数を任意に変更可能
      \end{itemize}
      
      \vspace{0.5em}
      \textbf{利点:} 実用性と安定性のバランスが優れている
    \end{column}
    \begin{column}{0.55\textwidth}
      \centering
      \includegraphics[width=\textwidth]{material/kiefer/output/golden_section_no_fixed_obs.png}
    \end{column}
  \end{columns}
\end{frame}

`
	outputPath := filepath.Join(OutputDir, TexName)
	return os.WriteFile(outputPath, []byte(content), 0644)
}
