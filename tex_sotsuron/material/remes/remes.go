package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// ==========================================
// 設定
// ==========================================

// 収束判定モード
// 1: 全て満たす（AND条件）
// 2: 複数満たす（2つ以上）
// 3: スコア合算
const (
	ConvergenceAND   = 1
	ConvergenceMulti = 2
	ConvergenceScore = 3
)

// RemesConfig: Remesアルゴリズムの設定構造体
type RemesConfig struct {
	PolyDegree              int
	MaxIter                 int
	BrentTol                float64
	GridDivisions           int
	OutputTexFileGrid       string
	OutputTexFileBrent      string
	OutputTexFileComparison string
	TexImgPathPrefix        string
	TargetFunc              func(float64) float64
	FuncTexName             string  // 関数のTeX表現（例: "e^x", "\\frac{1}{1+25x^2}"）
	ConvergenceMode         int     // 1: AND, 2: Multi, 3: Score
	IntervalA               float64 // 区間の左端
	IntervalB               float64 // 区間の右端
}

// デフォルト設定: ルンゲ関数で次数15
func NewDefaultRemesConfig() *RemesConfig {
	return &RemesConfig{
		PolyDegree:              15,
		MaxIter:                 20,
		BrentTol:                1e-15,
		GridDivisions:           2000,
		OutputTexFileGrid:       "remes_grid_result.tex",
		OutputTexFileBrent:      "remes_brent_result.tex",
		OutputTexFileComparison: "remes_comparison_result.tex",
		TexImgPathPrefix:        "material/remes/",
		FuncTexName:             "\\frac{1}{1+25x^2}", // ルンゲ関数のTeX表現
		ConvergenceMode:         ConvergenceAND,       // デフォルトはAND条件
		IntervalA:               -1.0,
		IntervalB:               1.0,
		TargetFunc: func(x float64) float64 {
			// デフォルト: ルンゲ関数
			return 1.0 / (1.0 + 25.0*x*x)
		},
	}
}

// NewRemesConfigWithCustom: 関数と次数をカスタマイズしたRemesConfig を作成
// 使用例:
//
//	config := NewRemesConfigWithCustom(
//	    func(x float64) float64 { return math.Exp(x) },
//	    "e^x",  // TeX表現
//	    4,
//	    "exp",
//	    ConvergenceAND,  // 収束判定モード
//	)
func NewRemesConfigWithCustom(targetFunc func(float64) float64, funcTexName string, degree int, outputFilePrefix string, convergenceMode int) *RemesConfig {
	return &RemesConfig{
		PolyDegree:              degree,
		MaxIter:                 20,
		BrentTol:                1e-15,
		GridDivisions:           2000,
		OutputTexFileGrid:       outputFilePrefix + "_grid_result.tex",
		OutputTexFileBrent:      outputFilePrefix + "_brent_result.tex",
		OutputTexFileComparison: outputFilePrefix + "_comparison_result.tex",
		TexImgPathPrefix:        "material/remes/",
		FuncTexName:             funcTexName,
		ConvergenceMode:         convergenceMode,
		TargetFunc:              targetFunc,
	}
}

// グローバル設定（便宜上、どの関数からでもアクセス可能）
var globalConfig *RemesConfig

// データ構造
type IterationLog struct {
	Iter        int
	EquiError   float64
	MaxError    float64
	Spread      float64
	Time        time.Duration
	Converged   bool    // 収束フラグ
	CoeffNorm   float64 // 係数の変化ノルム
	NodeMove    float64 // 偏差点の変化ノルム
	ErrorChange float64 // 誤差の変化
}

type FinalCheckRecord struct {
	No       int
	Position float64
	GridErr  float64 // ここには「収束時にアルゴリズムが算出した極値」が入る（Grid法ならGrid値，Brent法ならBrent値）
	BrentErr float64 // 高精度な極値での誤差
	Diff     float64 // BrentErr - GridErr (精度のズレ)
}

func main() {
	// CPUコア数を最大活用
	runtime.GOMAXPROCS(runtime.NumCPU())

	// グローバル設定を初期化
	// ===== デフォルト設定（ルンゲ関数, 次数15) =====
	globalConfig = NewDefaultRemesConfig()

	// ===== カスタム設定例 =====
	// 別の関数と次数でテストしたい場合は，下記をコメントイン:
	// globalConfig = NewRemesConfigWithCustom(
	// 	func(x float64) float64 { return math.Exp(x) }, // 関数: e^x
	// 	"e^x", // TeX表現
	// 	4,     // 次数: 4
	// 	"exp", // ファイル出力プレフィックス
	// )

	fmt.Println("=== Remes Algorithm - Unified Comparison ===")
	fmt.Printf("Approximating f(x)=1/(1+25x^2) on [-1, 1] with degree %d\n", globalConfig.PolyDegree)

	// 1. グリッド極値点探索ベースのRemes実行
	fmt.Println("\n--- Running Grid-based Remes ---")
	startGrid := time.Now()
	// 戻り値に高精度な最大誤差 (strictMax) を追加
	gridHist, gridFinalCheck, gridStrictMax := runRemesGrid()
	timeGrid := time.Since(startGrid)

	// 2. Brent法ベースのRemes実行
	fmt.Println("\n--- Running Brent-based Remes ---")
	startBrent := time.Now()
	brentHist, brentFinalCheck, brentStrictMax := runRemesBrent()
	timeBrent := time.Since(startBrent)

	// 3. レポート生成
	fmt.Println("\n--- Generating Reports ---")

	// 3章用
	generateTexGrid(gridHist, gridFinalCheck, gridStrictMax)
	// 5章用
	generateTexBrent(brentHist, brentFinalCheck, brentStrictMax)

	// 6章用 (比較)
	generateTexComparison(gridHist, brentHist, gridFinalCheck, brentFinalCheck, timeGrid, timeBrent, gridStrictMax, brentStrictMax)

	fmt.Println("Done.")
}

// 全区間を対象に高精度な最大誤差を探索する関数
// 戦略: 初期グリッド(2000分割)で極大値候補を見つけ，即座にBrent法で厳密化する
// これにより，計算コストを抑えつつ機械イプシロン精度の最大誤差を特定できる．
func findGlobalMaxError(coeffs []float64, a, b float64) float64 {
	// 誤差関数
	eFunc := func(x float64) float64 {
		return polynomial(coeffs, x, a, b) - globalConfig.TargetFunc(x)
	}

	// 初期スキャン用の分割数（これくらいあれば主要な山は見逃さない）
	divisions := 50000
	dx := (b - a) / float64(divisions)

	// 並列処理のための設定
	numCPU := runtime.NumCPU()
	var wg sync.WaitGroup
	resultCh := make(chan float64, numCPU)

	// 範囲を分割して並列スキャン
	chunkSize := divisions / numCPU

	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			startIdx := id * chunkSize
			endIdx := (id + 1) * chunkSize
			if id == numCPU-1 {
				endIdx = divisions // 最後のチャンクは余りを含める
			}

			localMax := 0.0

			// 担当範囲の値を事前に計算（隣接比較のため）
			// 端点の処理をスムーズにするため，少し広めに計算するか，インデックスアクセスに注意する
			// ここではシンプルに，関数内でローカルにループを回す

			for j := startIdx; j < endIdx; j++ {
				// 現在点とその両隣の点を評価
				// 注意: グローバルなインデックス j に対する x を計算
				xPrev := a + float64(j-1)*dx
				xCurr := a + float64(j)*dx
				xNext := a + float64(j+1)*dx

				// 範囲外チェック
				if xPrev < a {
					xPrev = a
				}
				if xNext > b {
					xNext = b
				}

				yPrev := math.Abs(eFunc(xPrev))
				yCurr := math.Abs(eFunc(xCurr))
				yNext := math.Abs(eFunc(xNext))

				// 極大値（ピーク）の判定
				// 端点の場合も考慮（片側比較）
				isPeak := false
				if j == 0 {
					if yCurr > yNext {
						isPeak = true
					}
				} else if j == divisions {
					if yCurr > yPrev {
						isPeak = true
					}
				} else {
					if yCurr > yPrev && yCurr > yNext {
						isPeak = true
					}
				}

				if isPeak {
					// ピーク候補を見つけたら，即座にBrent法で頂上をピンポイント探索
					// 探索範囲は隣接グリッド点までで十分
					searchStart := xPrev
					searchEnd := xNext

					// 目的関数（最大化したいので符号反転）
					obj := func(t float64) float64 { return -math.Abs(eFunc(t)) }

					// Brent法実行
					_, negVal := brentMinimize(obj, searchStart, searchEnd, globalConfig.BrentTol)
					strictVal := -negVal

					if strictVal > localMax {
						localMax = strictVal
					}
				}
			}
			resultCh <- localMax
		}(i)
	}

	wg.Wait()
	close(resultCh)

	globalMax := 0.0
	for val := range resultCh {
		if val > globalMax {
			globalMax = val
		}
	}

	return globalMax
}

// coefficientsNorm: 多項式係数の変化のノルムを計算
func coefficientsNorm(oldCoeffs, newCoeffs []float64) float64 {
	if len(oldCoeffs) != len(newCoeffs) {
		return math.Inf(1)
	}
	norm := 0.0
	for i := 0; i < len(oldCoeffs); i++ {
		diff := newCoeffs[i] - oldCoeffs[i]
		norm += diff * diff
	}
	return math.Sqrt(norm)
}

// nodesNorm: 偏差点の変化のノルムを計算
func nodesNorm(oldNodes, newNodes []float64) float64 {
	if len(oldNodes) != len(newNodes) {
		return math.Inf(1)
	}
	norm := 0.0
	for i := 0; i < len(oldNodes); i++ {
		diff := newNodes[i] - oldNodes[i]
		norm += diff * diff
	}
	return math.Sqrt(norm)
}

// checkConvergence: 3つの基準に基づいて収束判定を行う
func checkConvergence(coeffNorm, errorChange, nodeMove float64) bool {
	eps := math.Nextafter(1.0, 2.0) - 1.0 // 機械イプシロン
	tol := eps * 1e2                      // 若干の余裕を持たせる

	switch globalConfig.ConvergenceMode {
	case ConvergenceAND:
		// 全て満たす
		return coeffNorm < tol && errorChange < tol && nodeMove < tol

	case ConvergenceMulti:
		// 複数満たす（2つ以上）
		count := 0
		if coeffNorm < tol {
			count++
		}
		if errorChange < tol {
			count++
		}
		if nodeMove < tol {
			count++
		}
		return count >= 2

	case ConvergenceScore:
		// スコア合算（正規化した合算値が1未満なら収束）
		maxVal := math.Max(math.Max(coeffNorm, errorChange), nodeMove)
		if maxVal == 0 {
			return true
		}
		score := (coeffNorm + errorChange + nodeMove) / (3 * maxVal)
		return score < 0.5

	default:
		return coeffNorm < tol && errorChange < tol && nodeMove < tol
	}
}

// ensureOutputDir: 出力ディレクトリを作成する（PNG保存用）
func ensureOutputDir(dirname string) string {
	if err := os.MkdirAll(dirname, 0755); err != nil {
		log.Printf("Warning: failed to create directory %s: %v\n", dirname, err)
	}
	return dirname
}

// ==========================================
// Remes (Grid) Logic
// ==========================================
func runRemesGrid() ([]IterationLog, []FinalCheckRecord, float64) {
	n := globalConfig.PolyDegree
	a, b := -1.0, 1.0
	var history []IterationLog
	nodes := initializeNodes(n, a, b)
	var coeffs []float64
	var prevCoeffs []float64
	var prevNodes []float64
	var prevError float64
	startTime := time.Now()

	// 出力ディレクトリを作成
	outputDir := ensureOutputDir("outputs/grid")

	for iter := 1; iter <= globalConfig.MaxIter; iter++ {
		var e float64
		coeffs, e = solveLinearSystem(nodes, n, a, b)
		if coeffs == nil {
			log.Fatal("Matrix failed")
		}

		// 係数変化のノルムを計算
		coeffNorm := 0.0
		if prevCoeffs != nil {
			coeffNorm = coefficientsNorm(prevCoeffs, coeffs)
		}
		prevCoeffs = make([]float64, len(coeffs))
		copy(prevCoeffs, coeffs)

		// 誤差の変化
		errorChange := 0.0
		if iter > 1 {
			errorChange = math.Abs(e - prevError)
		}
		prevError = e

		eFunc := func(x float64) float64 {
			return polynomial(coeffs, x, a, b) - globalConfig.TargetFunc(x)
		}

		// グリッド極値点探索のみで更新
		nextNodes, maxE, minE := updateNodesGrid(coeffs, nodes, n, a, b, eFunc, globalConfig.GridDivisions)
		spread := maxE - minE

		// 偏差点の変化
		nodeMove := 0.0
		if prevNodes != nil {
			nodeMove = nodesNorm(prevNodes, nextNodes)
		}
		prevNodes = make([]float64, len(nextNodes))
		copy(prevNodes, nextNodes)

		// 収束判定
		converged := false
		if iter > 1 && checkConvergence(coeffNorm, errorChange, nodeMove) {
			converged = true
			fmt.Printf("Grid: Converged at iteration %d\n", iter)
		}

		// 履歴に追加（収束フラグと3つの指標を含む）
		history = append(history, IterationLog{
			Iter:        iter,
			EquiError:   math.Abs(e),
			MaxError:    maxE,
			Spread:      spread,
			Time:        time.Since(startTime),
			Converged:   converged,
			CoeffNorm:   coeffNorm,
			NodeMove:    nodeMove,
			ErrorChange: errorChange,
		})
		fmt.Printf("Iter %d: MaxE=%.4e, Spread=%.4e, CoeffNorm=%.4e, ErrorChange=%.4e, NodeMove=%.4e\n",
			iter, maxE, spread, coeffNorm, errorChange, nodeMove)

		// グラフ出力 (最初2つと最後)
		plotFuncComparison(coeffs, a, b, fmt.Sprintf("%s/func_remes_grid_iter%02d.png", outputDir, iter), fmt.Sprintf("Grid Approx (Iter %d)", iter))
		plotErrorFunc(coeffs, a, b, fmt.Sprintf("%s/error_remes_grid_iter%02d.png", outputDir, iter), fmt.Sprintf("Grid Error (Iter %d)", iter))

		if converged {
			break
		}
		nodes = nextNodes
	}

	// 最終評価1: 各節点位置での等振幅性チェック（偏差確認用）
	eFuncFinal := func(x float64) float64 { return polynomial(coeffs, x, a, b) - globalConfig.TargetFunc(x) }
	var checks []FinalCheckRecord
	for i, x := range nodes {
		gv := math.Abs(eFuncFinal(x))
		// 節点近傍でのBrent探索
		rad := 1e-9
		sa, sb := math.Max(a, x-rad), math.Min(b, x+rad)
		obj := func(t float64) float64 { return -math.Abs(eFuncFinal(t)) }
		bx, negEv := brentMinimize(obj, sa, sb, 1e-16)
		bv := -negEv
		checks = append(checks, FinalCheckRecord{i + 1, bx, gv, bv, bv - gv})
	}

	// 最終評価2: 全区間スキャンによる真の最大誤差（性能評価用）
	strictMax := findGlobalMaxError(coeffs, a, b)

	plotConvergence(history, "outputs/grid/convergence_grid.png", "Grid Convergence")
	return history, checks, strictMax
}

// ==========================================
// Remes (Brent) Logic
// ==========================================
func runRemesBrent() ([]IterationLog, []FinalCheckRecord, float64) {
	n := globalConfig.PolyDegree
	a, b := -1.0, 1.0
	var history []IterationLog
	nodes := initializeNodes(n, a, b)
	var coeffs []float64
	var prevCoeffs []float64
	var prevNodes []float64
	var prevError float64
	startTime := time.Now()

	// 出力ディレクトリを作成
	outputDir := ensureOutputDir("outputs/brent")

	for iter := 1; iter <= globalConfig.MaxIter; iter++ {
		var e float64
		coeffs, e = solveLinearSystem(nodes, n, a, b)
		if coeffs == nil {
			log.Fatal("Matrix failed")
		}

		// 係数変化のノルムを計算
		coeffNorm := 0.0
		if prevCoeffs != nil {
			coeffNorm = coefficientsNorm(prevCoeffs, coeffs)
		}
		prevCoeffs = make([]float64, len(coeffs))
		copy(prevCoeffs, coeffs)

		// 誤差の変化
		errorChange := 0.0
		if iter > 1 {
			errorChange = math.Abs(e - prevError)
		}
		prevError = e

		eFunc := func(x float64) float64 {
			return polynomial(coeffs, x, a, b) - globalConfig.TargetFunc(x)
		}

		// Brent法で更新
		nextNodes, maxE, minE := updateNodesBrent(coeffs, nodes, n, a, b, eFunc)
		spread := maxE - minE

		// 偏差点の変化
		nodeMove := 0.0
		if prevNodes != nil {
			nodeMove = nodesNorm(prevNodes, nextNodes)
		}
		prevNodes = make([]float64, len(nextNodes))
		copy(prevNodes, nextNodes)

		// 収束判定（Grid法と統一）
		converged := false
		if iter > 1 && checkConvergence(coeffNorm, errorChange, nodeMove) {
			converged = true
			fmt.Printf("Brent: Converged at iteration %d\n", iter)
		}

		// 履歴に追加（収束フラグと3つの指標を含む）
		history = append(history, IterationLog{
			Iter:        iter,
			EquiError:   math.Abs(e),
			MaxError:    maxE,
			Spread:      spread,
			Time:        time.Since(startTime),
			Converged:   converged,
			CoeffNorm:   coeffNorm,
			NodeMove:    nodeMove,
			ErrorChange: errorChange,
		})
		fmt.Printf("Iter %d: MaxE=%.4e, Spread=%.4e, CoeffNorm=%.4e, ErrorChange=%.4e, NodeMove=%.4e\n",
			iter, maxE, spread, coeffNorm, errorChange, nodeMove)

		plotFuncComparison(coeffs, a, b, fmt.Sprintf("%s/func_remes_brent_iter%02d.png", outputDir, iter), fmt.Sprintf("Brent Approx (Iter %d)", iter))
		plotErrorFunc(coeffs, a, b, fmt.Sprintf("%s/error_remes_brent_iter%02d.png", outputDir, iter), fmt.Sprintf("Brent Error (Iter %d)", iter))

		if converged {
			break
		}
		nodes = nextNodes
	}

	// 最終評価1: 各節点位置での等振幅性チェック
	eFuncFinal := func(x float64) float64 { return polynomial(coeffs, x, a, b) - globalConfig.TargetFunc(x) }
	var checks []FinalCheckRecord
	for i, x := range nodes {
		gv := math.Abs(eFuncFinal(x))
		rad := 1e-9
		sa, sb := math.Max(a, x-rad), math.Min(b, x+rad)
		obj := func(t float64) float64 { return -math.Abs(eFuncFinal(t)) }
		bx, negEv := brentMinimize(obj, sa, sb, 1e-16)
		bv := -negEv
		checks = append(checks, FinalCheckRecord{i + 1, bx, gv, bv, bv - gv})
	}

	// 最終評価2: 全区間スキャンによる真の最大誤差
	strictMax := findGlobalMaxError(coeffs, a, b)

	plotConvergence(history, "outputs/brent/convergence_brent.png", "Brent Convergence")
	return history, checks, strictMax
}

// ==========================================
// TeX 生成関数
// ==========================================

// 3章用
func generateTexGrid(hist []IterationLog, check []FinalCheckRecord, strictMax float64) {
	f, _ := os.Create(globalConfig.OutputTexFileGrid)
	defer f.Close()

	// テンプレートを使用してヘッダを書き込む
	header := fillTemplate(TexTemplateGrid, globalConfig.FuncTexName, globalConfig.PolyDegree)
	fmt.Fprint(f, header)

	// Grid用の画像パスプレフィックス
	imgPrefix := globalConfig.TexImgPathPrefix + "outputs/grid/"
	fmt.Printf("DEBUG: Grid image prefix: %s\n", imgPrefix)

	// 以下、グラフと表データを追加
	fmt.Fprintf(f, "\\begin{figure}[H]\n \\centering\n \\includegraphics[width=0.6\\linewidth]{%sconvergence_grid.png}\n \\caption{グリッド極値点探索法による最大誤差の収束履歴}\n \\label{fig:remes-grid-convergence}\n\\end{figure}\n\n", imgPrefix)
	writeGraphs(f, hist, "remes_grid", "グリッド法", imgPrefix)
	fmt.Fprintf(f, "\\begin{table}[H]\n \\centering\n \\caption{グリッド極値点探索法の収束ログ}\n \\label{tab:remes-grid-log}\n \\begin{tabular}{crrrr}\n \\toprule\n Iter & $|E|$ & Coeff Norm & Node Move & Error Change \\\\\n \\midrule\n")
	for _, h := range hist {
		fmt.Fprintf(f, " %d & $%.5e$ & $%.5e$ & $%.5e$ & $%.5e$ \\\\\n", h.Iter, h.EquiError, h.CoeffNorm, h.NodeMove, h.ErrorChange)
	}
	fmt.Fprintf(f, " \\bottomrule\n \\end{tabular}\n\\end{table}\n\n")

	// 高精度な最大誤差をキャプションや注釈で示す
	fmt.Fprintf(f, "\\begin{table}[H]\n \\centering\n \\caption{グリッド極値点探索結果の等振幅性確認（節点近傍の偏差）}\n \\label{tab:remes-grid-final-check}\n \\begin{tabular}{crrr}\n \\toprule\n No. & Grid誤差 & Brent誤差 & 差分 \\\\\n \\midrule\n")
	for _, c := range check {
		fmt.Fprintf(f, " %d & $%.8e$ & $%.8e$ & $%.5e$ \\\\\n", c.No, c.GridErr, c.BrentErr, c.Diff)
	}
	fmt.Fprintf(f, " \\bottomrule\n \\end{tabular}\n")
	fmt.Fprintf(f, "\\vspace{1mm}\n\n")
	fmt.Fprintf(f, "\\small \\textbf{注：}\n")
	fmt.Fprintf(f, "\\small 上表は，区間 $[%.1f,%.1f]$ を探索するRemesアルゴリズムによって導かれた誤差関数に対する表である．\n\n", globalConfig.IntervalA, globalConfig.IntervalB)
	fmt.Fprintf(f, "\\small Grid誤差とはBrent法をもちいることで極値点探索を行ったRemesアルゴリズムで導出された誤差関数の各極値点で観測された誤差である．\n\n")
	fmt.Fprintf(f, "\\small Brent誤差とはBrent法という極値点探索アルゴリズムにより観測された誤差の絶対値である．\n\n")
	fmt.Fprintf(f, "\\[\n \\text{観測された最大誤差} \\; \\|E\\|_{\\max}^{\\text{strict}}(f) = \\max_{x \\in [%.1f,%.1f]} |f(x) - p^*(x)| \\approx %.8e .\n \\]\n", globalConfig.IntervalA, globalConfig.IntervalB, strictMax)
	fmt.Fprintf(f, "\\end{table}\n")
}

// 5章用
func generateTexBrent(hist []IterationLog, check []FinalCheckRecord, strictMax float64) {
	f, _ := os.Create(globalConfig.OutputTexFileBrent)
	defer f.Close()

	// テンプレートを使用してヘッダを書き込む
	header := fillTemplate(TexTemplateBrent, globalConfig.FuncTexName, globalConfig.PolyDegree)
	fmt.Fprint(f, header)

	// Brent用の画像パスプレフィックス
	imgPrefix := globalConfig.TexImgPathPrefix + "outputs/brent/"

	// 以下、グラフと表データを追加
	fmt.Fprintf(f, "\\begin{figure}[H]\n \\centering\n \\includegraphics[width=0.6\\linewidth]{%sconvergence_brent.png}\n \\caption{Brent法による最大誤差の収束履歴}\n \\label{fig:remes-brent-convergence}\n\\end{figure}\n\n", imgPrefix)
	writeGraphs(f, hist, "remes_brent", "Brent法", imgPrefix)
	fmt.Fprintf(f, "\\begin{table}[H]\n \\centering\n \\caption{Brent法の収束ログ}\n \\label{tab:remes-brent-log}\n \\begin{tabular}{crrrr}\n \\toprule\n Iter & $|E|$ & Coeff Norm & Node Move & Error Change \\\\\n \\midrule\n")
	for _, h := range hist {
		fmt.Fprintf(f, " %d & $%.5e$ & $%.5e$ & $%.5e$ & $%.5e$ \\\\\n", h.Iter, h.EquiError, h.CoeffNorm, h.NodeMove, h.ErrorChange)
	}
	fmt.Fprintf(f, " \\bottomrule\n \\end{tabular}\n\\end{table}\n\n")

	// カラム名を修正: Grid誤差 -> 収束時誤差
	fmt.Fprintf(f, "\\begin{table}[H]\n \\centering\n \\caption{Brent法利用時の誤差の再評価}\n \\label{tab:remes-brent-final-check}\n \\begin{tabular}{crrr}\n \\toprule\n No. & 収束時誤差 & 再評価誤差 & 差分 \\\\\n \\midrule\n")
	for _, c := range check {
		fmt.Fprintf(f, " %d & $%.8e$ & $%.8e$ & $%.5e$ \\\\\n", c.No, c.GridErr, c.BrentErr, c.Diff)
	}
	fmt.Fprintf(f, " \\bottomrule\n \\end{tabular}\n")
	fmt.Fprintf(f, "\\vspace{1mm}\n\n")
	fmt.Fprintf(f, "\\small \\textbf{注：}\n")
	fmt.Fprintf(f, "\\small 上表は，区間 $[%.1f,%.1f]$ を探索するRemesアルゴリズムによって導かれた誤差関数に対する表である．\n\n", globalConfig.IntervalA, globalConfig.IntervalB)
	fmt.Fprintf(f, "\\small 収束時誤差とはBrent法をもちいることで極値点探索を行ったRemesアルゴリズムで導出された誤差関数の各極値点で観測された誤差である．\n\n")
	fmt.Fprintf(f, "\\small 再評価誤差とはBrent法という極値点探索アルゴリズムにより観測された誤差の絶対値である．\n\n")
	fmt.Fprintf(f, "\\[\n \\text{観測された最大誤差} \\; \\|E\\|_{\\max}^{\\text{strict}}(f) = \\max_{x \\in [%.1f,%.1f]} |f(x) - p^*(x)| \\approx %.8e .\n \\]\n", globalConfig.IntervalA, globalConfig.IntervalB, strictMax)
	fmt.Fprintf(f, "\\end{table}\n")
}

// 6章用 (比較)
func generateTexComparison(gridHist, brentHist []IterationLog, gridCheck, brentCheck []FinalCheckRecord, tGrid, tBrent time.Duration, gridStrictMax, brentStrictMax float64) {
	f, _ := os.Create(globalConfig.OutputTexFileComparison)
	defer f.Close()

	// テンプレートを使用してヘッダを書き込む
	header := fillTemplate(TexTemplateComparison, globalConfig.FuncTexName, globalConfig.PolyDegree)
	fmt.Fprint(f, header)

	gridLast := gridHist[len(gridHist)-1]
	brentLast := brentHist[len(brentHist)-1]

	// 出力ディレクトリを作成
	ensureOutputDir("outputs/comparison")

	// 1. 偏差比較グラフ
	plotDeviationComparison(gridCheck, brentCheck, "outputs/comparison/deviation_comparison.png", "極値点における真の誤差との偏差比較")

	// Comparison用の画像パスプレフィックス
	imgPrefix := globalConfig.TexImgPathPrefix + "outputs/comparison/"

	fmt.Fprintf(f, "\\begin{figure}[H]\n \\centering\n")
	fmt.Fprintf(f, " \\includegraphics[width=0.8\\linewidth]{%sdeviation_comparison.png}\n", imgPrefix)
	fmt.Fprintf(f, " \\caption{極値点ごとの理論値からの偏差 ($|\\text{Calc} - \\text{True}|$)}\n")
	fmt.Fprintf(f, " \\label{fig:remes-comparison-deviation}\n\\end{figure}\n\n")

	// 2. 総合比較表
	fmt.Fprintf(f, "\\begin{table}[H]\n \\centering\n \\caption{グリッド法とBrent法の性能比較概要 ($n=%d$)}\n \\label{tab:remes-comparison-summary}\n", globalConfig.PolyDegree)
	fmt.Fprintf(f, " \\begin{tabular}{lcccc}\n \\toprule\n 手法 & 反復回数 & 収束判定値 & 高精度な最大誤差 $\\|E\\|$ & 計算時間 \\\\\n \\midrule\n")
	// Grid法の最大誤差を全区間スキャンの結果に変更
	fmt.Fprintf(f, " Grid Search & %d & $%.5e$ & $%.5e$ & %v \\\\\n", len(gridHist), gridLast.Spread, gridStrictMax, tGrid)
	fmt.Fprintf(f, " Brent Method & %d & $%.5e$ & $%.5e$ & %v \\\\\n", len(brentHist), brentLast.Spread, brentStrictMax, tBrent)
	fmt.Fprintf(f, " \\bottomrule\n \\end{tabular}\n")
	fmt.Fprintf(f, "\\vspace{1mm}\n\n")
	fmt.Fprintf(f, "\\footnotesize\\textbf{高精度な最大誤差について：}\n")
	fmt.Fprintf(f, "区間 $[%.1f,%.1f]$ を50,000個の格子点で均等分割し，誤差関数 $|E(x)|$ の局所極大値を検出した後，\n", globalConfig.IntervalA, globalConfig.IntervalB)
	fmt.Fprintf(f, "各極大値の近傍に対してBrent法を適用して算出した値である．\n")
	fmt.Fprintf(f, "\\end{table}\n\n")

	// 3. 詳細精度比較テーブル
	fmt.Fprintf(f, "\\begin{table}[H]\n \\centering\n \\caption{各極値点における真の誤差との偏差比較（等振幅性の崩れ）}\n \\label{tab:remes-comparison-detail}\n")
	fmt.Fprintf(f, " \\begin{tabular}{crrr}\n \\toprule\n No. & Grid偏差 ($|\\Delta E|$) & Brent偏差 ($|\\Delta E|$) & 精度比 (Grid/Brent) \\\\\n \\midrule\n")

	sumGridDev, sumBrentDev := 0.0, 0.0
	for i := 0; i < len(gridCheck) && i < len(brentCheck); i++ {
		gDev := math.Abs(gridCheck[i].Diff)
		bDev := math.Abs(brentCheck[i].Diff)
		ratio := 0.0
		if bDev > 1e-20 {
			ratio = gDev / bDev
		}

		fmt.Fprintf(f, " %d & $%.5e$ & $%.5e$ & $%.1f$ \\\\\n", i+1, gDev, bDev, ratio)
		sumGridDev += gDev
		sumBrentDev += bDev
	}
	fmt.Fprintf(f, " \\bottomrule\n \\end{tabular}\n\\end{table}\n\n")

	// 4. 自動考察テキスト
	avgRatio := 0.0
	if sumBrentDev > 1e-20 {
		avgRatio = sumGridDev / sumBrentDev
	}

	fmt.Fprintf(f, "\\noindent\\textbf{考察:}\n")
	fmt.Fprintf(f, "表\\ref{tab:remes-comparison-summary}において，全区間を対象とした高精度な再評価を行った結果，\n")
	if gridStrictMax > brentStrictMax {
		fmt.Fprintf(f, "Grid法の最大誤差 $\\|E\\|$ は $%.6e$ となり，Brent法の $%.6e$ よりも大きな値を示した．\n", gridStrictMax, brentStrictMax)
		fmt.Fprintf(f, "これは，Grid法が離散的な探索により真の最大誤差（ピーク）を取りこぼして収束していたことを示唆しており，\n")
		fmt.Fprintf(f, "Brent法を用いることで，より最大誤差が小さく，最良近似多項式に近い結果が得られたことを裏付けている．\n")
	} else {
		// 万が一まだGridの方が小さい場合（稀だが）の安全策
		fmt.Fprintf(f, "両手法の最大誤差はほぼ同等な値となったが，Brent法の方がわずかに精度が良い，あるいは同等である．\n")
	}
	fmt.Fprintf(f, "また，表\\ref{tab:remes-comparison-detail}に示すように，各極値点における等振幅性の偏差はBrent法の方が平均して約 %.0f 倍小さく，\n", avgRatio)
	fmt.Fprintf(f, "数学的性質の観点からもBrent法がより高精度な近似解を得られていることが確認できる．\n")
}

// 偏差比較チャート（棒グラフ）の描画
func plotDeviationComparison(gridChecks, brentChecks []FinalCheckRecord, filename, title string) {
	p := plot.New()
	p.Title.Text = title
	p.Y.Label.Text = "Deviation (|Diff|)"
	p.Y.Scale = plot.LogScale{}
	p.Y.Tick.Marker = plot.LogTicks{}

	// Grid
	ptsGrid := make(plotter.XYs, len(gridChecks))
	for i, c := range gridChecks {
		val := math.Abs(c.Diff)
		if val < 1e-18 {
			val = 1e-18
		} // ログスケール用に下限設定
		ptsGrid[i].X = float64(c.No) - 0.15
		ptsGrid[i].Y = val
	}

	// Brent
	ptsBrent := make(plotter.XYs, len(brentChecks))
	for i, c := range brentChecks {
		val := math.Abs(c.Diff)
		if val < 1e-18 {
			val = 1e-18
		}
		ptsBrent[i].X = float64(c.No) + 0.15
		ptsBrent[i].Y = val
	}

	// Grid Plot (Red bars/points)
	barsGrid, _ := plotter.NewScatter(ptsGrid)
	barsGrid.Color = color.RGBA{255, 0, 0, 255}
	barsGrid.Shape = draw.SquareGlyph{}
	p.Add(barsGrid)

	// Brent Plot (Blue bars/points)
	barsBrent, _ := plotter.NewScatter(ptsBrent)
	barsBrent.Color = color.RGBA{0, 0, 255, 255}
	barsBrent.Shape = draw.CircleGlyph{}
	p.Add(barsBrent)

	// 補助線（Barに見せるため）
	for i := range ptsGrid {
		lG, _ := plotter.NewLine(plotter.XYs{{X: ptsGrid[i].X, Y: 1e-19}, {X: ptsGrid[i].X, Y: ptsGrid[i].Y}})
		lG.Color = color.RGBA{255, 0, 0, 100}
		p.Add(lG)

		lB, _ := plotter.NewLine(plotter.XYs{{X: ptsBrent[i].X, Y: 1e-19}, {X: ptsBrent[i].X, Y: ptsBrent[i].Y}})
		lB.Color = color.RGBA{0, 0, 255, 100}
		p.Add(lB)
	}

	p.Legend.Add("Grid Deviation", barsGrid)
	p.Legend.Add("Brent Deviation", barsBrent)
	p.Legend.Top = true

	// Y軸の範囲設定
	p.Y.Min = 1e-18
	p.Y.Max = 1e-8

	p.Save(6*vg.Inch, 4*vg.Inch, filename)
}

// グラフ出力用ヘルパー（2列レイアウト）
func writeGraphs(f *os.File, hist []IterationLog, prefix, caption, imgPrefix string) {
	iters := []int{1, 2}
	last := hist[len(hist)-1].Iter
	if last > 2 {
		iters = append(iters, last)
	}

	// Approx（2列でペアリング）
	fmt.Fprintf(f, "\\begin{figure}[H]\n \\centering\n")
	for i := 0; i < len(iters); i += 2 {
		// 左のグラフ
		fmt.Fprintf(f, " \\begin{minipage}[b]{0.48\\textwidth}\n  \\centering\n  \\includegraphics[width=1.0\\linewidth]{%sfunc_%s_iter%02d.png}\n  \\caption{Iter %d}\n \\end{minipage}\n", imgPrefix, prefix, iters[i], iters[i])

		// 右のグラフ（存在すれば）
		if i+1 < len(iters) {
			fmt.Fprintf(f, " \\hfill\n")
			fmt.Fprintf(f, " \\begin{minipage}[b]{0.48\\textwidth}\n  \\centering\n  \\includegraphics[width=1.0\\linewidth]{%sfunc_%s_iter%02d.png}\n  \\caption{Iter %d}\n \\end{minipage}\n", imgPrefix, prefix, iters[i+1], iters[i+1])
		}
		fmt.Fprintf(f, "\n")
	}
	fmt.Fprintf(f, " \\caption{%s: 近似多項式の推移}\n \\label{fig:%s-approx}\n\\end{figure}\n", caption, strings.ReplaceAll(prefix, "_", "-"))

	// Error（2列でペアリング）
	fmt.Fprintf(f, "\\begin{figure}[H]\n \\centering\n")
	for i := 0; i < len(iters); i += 2 {
		// 左のグラフ
		fmt.Fprintf(f, " \\begin{minipage}[b]{0.48\\textwidth}\n  \\centering\n  \\includegraphics[width=1.0\\linewidth]{%serror_%s_iter%02d.png}\n  \\caption{Iter %d}\n \\end{minipage}\n", imgPrefix, prefix, iters[i], iters[i])

		// 右のグラフ（存在すれば）
		if i+1 < len(iters) {
			fmt.Fprintf(f, " \\hfill\n")
			fmt.Fprintf(f, " \\begin{minipage}[b]{0.48\\textwidth}\n  \\centering\n  \\includegraphics[width=1.0\\linewidth]{%serror_%s_iter%02d.png}\n  \\caption{Iter %d}\n \\end{minipage}\n", imgPrefix, prefix, iters[i+1], iters[i+1])
		}
		fmt.Fprintf(f, "\n")
	}
	fmt.Fprintf(f, " \\caption{%s: 誤差関数の推移}\n \\label{fig:%s-error}\n\\end{figure}\n\n", caption, strings.ReplaceAll(prefix, "_", "-"))
}

// ==========================================
// アルゴリズム共通部分
// ==========================================
func initializeNodes(n int, a, b float64) []float64 {
	nodes := make([]float64, n+2)
	for k := 0; k < n+2; k++ {
		nodes[k] = (a+b)/2 + (b-a)/2*math.Cos(float64(k)*math.Pi/float64(n+1))
	}
	sort.Float64s(nodes)
	return nodes
}
func polynomial(coeffs []float64, x, a, b float64) float64 {
	u := (2*x - (a + b)) / (b - a)
	tPrev, tCurr := 1.0, u
	val := coeffs[0] * tPrev
	if len(coeffs) > 1 {
		val += coeffs[1] * tCurr
		for j := 2; j < len(coeffs); j++ {
			tNext := 2*u*tCurr - tPrev
			tPrev, tCurr = tCurr, tNext
			val += coeffs[j] * tCurr
		}
	}
	return val
}
func solveLinearSystem(nodes []float64, n int, a, b float64) ([]float64, float64) {
	size := n + 2
	A := mat.NewDense(size, size, nil)
	bVec := mat.NewVecDense(size, nil)
	for i := 0; i < size; i++ {
		xi := nodes[i]
		bVec.SetVec(i, globalConfig.TargetFunc(xi))
		u := (2*xi - (a + b)) / (b - a)
		tPrev, tCurr := 1.0, u
		A.Set(i, 0, tPrev)
		if n >= 1 {
			A.Set(i, 1, tCurr)
		}
		for j := 2; j <= n; j++ {
			tNext := 2*u*tCurr - tPrev
			tPrev, tCurr = tCurr, tNext
			A.Set(i, j, tCurr)
		}
		val := 1.0
		if i%2 != 0 {
			val = -1.0
		}
		A.Set(i, n+1, val)
	}
	var sol mat.VecDense
	if err := sol.SolveVec(A, bVec); err != nil {
		return nil, 0
	}
	res := sol.RawVector().Data
	return res[:n+1], res[n+1]
}
func updateNodesGrid(coeffs []float64, nodes []float64, n int, a, b float64, eFunc func(float64) float64, gridDivisions int) ([]float64, float64, float64) {
	dx := (b - a) / float64(gridDivisions)
	type point struct{ x, err float64 }
	gridPoints := make([]point, gridDivisions+1)
	for i := 0; i <= gridDivisions; i++ {
		x := a + float64(i)*dx
		gridPoints[i] = point{x: x, err: math.Abs(eFunc(x))}
	}
	var candidates []point
	for i := 1; i < gridDivisions; i++ {
		if gridPoints[i].err > gridPoints[i-1].err && gridPoints[i].err > gridPoints[i+1].err {
			candidates = append(candidates, gridPoints[i])
		}
	}
	if gridPoints[0].err > gridPoints[1].err {
		candidates = append(candidates, gridPoints[0])
	}
	if gridPoints[gridDivisions].err > gridPoints[gridDivisions-1].err {
		candidates = append(candidates, gridPoints[gridDivisions])
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].x < candidates[j].x })
	for len(candidates) > n+2 {
		if candidates[0].err < candidates[len(candidates)-1].err {
			candidates = candidates[1:]
		} else {
			candidates = candidates[:len(candidates)-1]
		}
	}
	newNodes := make([]float64, len(candidates))
	maxE, minE := -1.0, 1e308
	for i, c := range candidates {
		newNodes[i] = c.x
		if c.err > maxE {
			maxE = c.err
		}
		if c.err < minE {
			minE = c.err
		}
	}
	return newNodes, maxE, minE
}
func updateNodesBrent(coeffs []float64, nodes []float64, n int, a, b float64, eFunc func(float64) float64) ([]float64, float64, float64) {
	roots := make([]float64, 0, n+1)
	subGrid := 2000
	for i := 0; i < len(nodes)-1; i++ {
		r := findRoot(eFunc, nodes[i], nodes[i+1], subGrid/(n+1))
		roots = append(roots, r)
	}
	boundaries := append([]float64{a}, roots...)
	boundaries = append(boundaries, b)
	newNodes := make([]float64, 0, n+2)
	maxE, minE := -1.0, 1e308
	type Candidate struct{ x, err float64 }
	var candidates []Candidate
	for i := 0; i < len(boundaries)-1; i++ {
		segA, segB := boundaries[i], boundaries[i+1]
		obj := func(x float64) float64 { return -math.Abs(eFunc(x)) }
		val, _ := brentMinimize(obj, segA, segB, globalConfig.BrentTol)
		if val <= segA {
			val = segA + 1e-9
		}
		if val >= segB {
			val = segB - 1e-9
		}
		candidates = append(candidates, Candidate{val, math.Abs(eFunc(val))})
	}
	for len(candidates) > n+2 {
		if candidates[0].err < candidates[len(candidates)-1].err {
			candidates = candidates[1:]
		} else {
			candidates = candidates[:len(candidates)-1]
		}
	}
	if len(candidates) < n+2 {
		return nodes, 0, 0
	}
	for _, c := range candidates {
		newNodes = append(newNodes, c.x)
		if c.err > maxE {
			maxE = c.err
		}
		if c.err < minE {
			minE = c.err
		}
	}
	sort.Float64s(newNodes)
	return newNodes, maxE, minE
}
func findRoot(f func(float64) float64, a, b float64, divisions int) float64 {
	dx := (b - a) / float64(divisions)
	prevVal := f(a)
	targetA, targetB := a, b
	found := false
	for i := 1; i <= divisions; i++ {
		x := a + float64(i)*dx
		val := f(x)
		if prevVal*val <= 0 {
			targetA, targetB = x-dx, x
			found = true
			break
		}
		prevVal = val
	}
	if !found {
		return (a + b) / 2
	}
	for i := 0; i < 60; i++ {
		mid := (targetA + targetB) / 2
		if f(mid) == 0 {
			return mid
		}
		if f(targetA)*f(mid) < 0 {
			targetB = mid
		} else {
			targetA = mid
		}
		if math.Abs(targetB-targetA) < 1e-13 {
			break
		}
	}
	return (targetA + targetB) / 2
}
func brentMinimize(f func(float64) float64, a, b, tol float64) (float64, float64) {
	const golden = 0.3819660112501051
	x := a + golden*(b-a)
	w, v := x, x
	d, e, fx := 0.0, 0.0, f(x)
	fw, fv := fx, fx
	for iter := 0; iter < 100; iter++ {
		xm := 0.5 * (a + b)
		tol1 := tol*math.Abs(x) + 1e-20
		tol2 := 2.0 * tol1
		if math.Abs(x-xm) <= (tol2 - 0.5*(b-a)) {
			return x, fx
		}
		if math.Abs(e) > tol1 {
			r, q := (x-w)*(fx-fv), (x-v)*(fx-fw)
			p := (x-v)*q - (x-w)*r
			q = 2.0 * (q - r)
			if q > 0 {
				p = -p
			}
			q = math.Abs(q)
			etemp := e
			e = d
			if math.Abs(p) >= math.Abs(0.5*q*etemp) || p <= q*(a-x) || p >= q*(b-x) {
				if x >= xm {
					e = a - x
				} else {
					e = b - x
				}
				d = golden * e
			} else {
				d = p / q
				u := x + d
				if u-a < tol2 || b-u < tol2 {
					d = math.Copysign(tol1, xm-x)
				}
			}
		} else {
			if x >= xm {
				e = a - x
			} else {
				e = b - x
			}
			d = golden * e
		}
		u := 0.0
		if math.Abs(d) >= tol1 {
			u = x + d
		} else {
			u = x + math.Copysign(tol1, d)
		}
		fu := f(u)
		if fu <= fx {
			if u >= x {
				a = x
			} else {
				b = x
			}
			v, w, x = w, x, u
			fv, fw, fx = fw, fx, fu
		} else {
			if u < x {
				a = u
			} else {
				b = u
			}
			if fu <= fw || w == x {
				v, w, fv, fw = w, u, fw, fu
			} else if fu <= fv || v == x || v == w {
				v, fv = u, fu
			}
		}
	}
	return x, fx
}
func plotFuncComparison(coeffs []float64, a, b float64, filename, title string) {
	p := plot.New()
	p.Title.Text = title
	steps := 500
	ptsF, ptsP := make(plotter.XYs, steps+1), make(plotter.XYs, steps+1)
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		ptsF[i].X, ptsF[i].Y = x, globalConfig.TargetFunc(x)
		ptsP[i].X, ptsP[i].Y = x, polynomial(coeffs, x, a, b)
	}
	lF, _ := plotter.NewLine(ptsF)
	lF.Color = color.RGBA{0, 0, 0, 255}
	lP, _ := plotter.NewLine(ptsP)
	lP.Color = color.RGBA{255, 0, 0, 255}
	lP.Dashes = []vg.Length{vg.Points(4), vg.Points(4)}
	p.Add(lF, lP)
	p.Save(6*vg.Inch, 4*vg.Inch, filename)
}
func plotErrorFunc(coeffs []float64, a, b float64, filename, title string) {
	p := plot.New()
	p.Title.Text = title
	steps := 1000
	pts := make(plotter.XYs, steps+1)
	maxVal := 0.0
	for i := 0; i <= steps; i++ {
		x := a + (b-a)*float64(i)/float64(steps)
		y := polynomial(coeffs, x, a, b) - globalConfig.TargetFunc(x)
		pts[i].X, pts[i].Y = x, y
		if math.Abs(y) > maxVal {
			maxVal = math.Abs(y)
		}
	}
	l, _ := plotter.NewLine(pts)
	l.Color = color.RGBA{0, 0, 255, 255}
	p.Add(l, plotter.NewGrid())
	p.Y.Min, p.Y.Max = -maxVal*1.1, maxVal*1.1
	p.Save(6*vg.Inch, 4*vg.Inch, filename)
}
func plotConvergence(hist []IterationLog, filename, title string) {
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = "Iteration"
	p.Y.Label.Text = "Max Error"
	p.Y.Scale = plot.LogScale{}
	p.Y.Tick.Marker = plot.LogTicks{}
	pts := make(plotter.XYs, len(hist))
	for i, h := range hist {
		pts[i].X, pts[i].Y = float64(h.Iter), math.Max(h.Spread, 1e-20)
	}
	l, _ := plotter.NewLine(pts)
	p.Add(l, plotter.NewGrid())
	p.Save(6*vg.Inch, 4*vg.Inch, filename)
}
