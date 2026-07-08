package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	"gonum.org/v1/gonum/mat"
)

// main はプログラムの開始点（エントリーポイント）です．
func main() {
	fmt.Println("Remesアルゴリズムによる最良近似多項式の導出")
	remesAlgorithm()
}

// 既存の部品関数（initializeNodes, solveLinearSystem, errorFunction, findExtrema, checkConvergence）はそのまま利用

// 実質的なターゲット関数 f(x) を定義．
func targetFunction(x float64) float64 {
	return math.Cos(x)
}

// remesAlgorithm は，Remesアルゴリズム全体の流れを制御する司令塔です．
func remesAlgorithm() {
	// --- パラメータ設定 ---
	n := 3
	intervalStart, intervalEnd := -1.0, 1.0
	gridSize := 500
	maxLoop := 15
	tolerance := 1e-20
	brentTolerance := 1e-10

	// 初期節点（チェビシェフ）
	nodes := initializeNodes(n, intervalStart, intervalEnd)

	var coeffs []float64 // (c0..cn) を保持
	var e float64        // 均等誤差

	prevMaxErr := math.Inf(1) // 初回は差分評価できないため +Inf でマーク
	for i := 0; i < maxLoop; i++ {
		fmt.Printf("\n--- イテレーション %d ---\n", i+1)
		fmt.Printf("現在の参照点: %s\n", formatFloatSlice(nodes, 6))

		// 係数と誤差を解く
		coeffs, e = solveLinearSystem(nodes)
		if coeffs == nil {
			fmt.Println("計算を中断します．")
			return
		}
		fmt.Printf("係数: %s\n", formatFloatSlice(coeffs, 8))
		latexPoly := formatPolynomialLatex(coeffs, fmt.Sprintf("P_{%d,%d}", n, i+1), "x", 8, 1e-12, true)
		fmt.Printf("LaTeX多項式: %s\n", latexPoly)
		fmt.Printf("均等誤差 E: %.9f\n", e)

		// 誤差関数
		eFunc := errorFunction(coeffs)

		// 極値探索
		extrema := findExtrema(eFunc, n, gridSize, intervalStart, intervalEnd, brentTolerance)
		fmt.Printf("極値点候補: %s\n", formatFloatSlice(extrema, 6))

		// 収束判定（最大誤差の前回との差に注目）
		converged, maxErr := checkConvergence(prevMaxErr, eFunc, extrema, tolerance)
		if converged {
			fmt.Println("\n収束しました（最大誤差差分が許容範囲内）．")
			break
		}
		prevMaxErr = maxErr
		nodes = extrema
		if i == maxLoop-1 {
			fmt.Println("\n最大反復回数に到達（未収束）．")
		}
	}
}

// 次数と端点を入力することでn+2個の初期参照点を生成する．
func initializeNodes(n int, a, b float64) []float64 {
	fmt.Println("ステップ1: 初期参照点をチェビシェフ節点で生成中...")
	nodes := make([]float64, n+2)
	for k := 0; k < n+2; k++ {
		nodes[k] = (a+b)/2.0 + (b-a)/2.0*math.Cos(float64(k)*math.Pi/float64(n+1))
	}
	sort.Float64s(nodes) // 昇順にソート
	return nodes
}

// 参照点列から連立方程式を解き，係数と誤差Eを出力する．
func solveLinearSystem(nodes []float64) (coeffs []float64, e float64) {
	m := len(nodes)
	if m < 2 {
		log.Println("参照点が不足しています．")
		return nil, 0.0
	}

	A := mat.NewDense(m, m, nil)
	bData := make([]float64, m)

	for i := 0; i < m; i++ {
		xi := nodes[i]
		bData[i] = targetFunction(xi) // ベクトルbの構築
		for j := 0; j < m; j++ {      // 行列Aの構築
			if j < m-1 {
				A.Set(i, j, math.Pow(xi, float64(j)))
			} else {
				A.Set(i, m-1, -math.Pow(-1, float64(i)))
			}
		}
	}
	b := mat.NewVecDense(m, bData)

	var c mat.VecDense
	if err := c.SolveVec(A, b); err != nil {
		log.Printf("連立方程式の解決に失敗しました: %v", err)
		return nil, 0.0
	}

	solution := c.RawVector().Data
	coeffs = solution[:m-1]
	e = solution[m-1]
	return coeffs, e
}

// 係数行列とx座標をもとに 多項式 P(x) を作成．
func polynomial(coeffs []float64, x float64) float64 {
	result := 0.0
	for i := len(coeffs) - 1; i >= 0; i-- {
		result = result*x + coeffs[i]
	}
	return result
}

// 多項式とターゲット関数の差を誤差関数として返す．
func errorFunction(coeffs []float64) func(float64) float64 {
	return func(x float64) float64 {
		return polynomial(coeffs, x) - targetFunction(x)
	}
}

// 誤差関数をもとに極値点を探索する．
// ---------------------------------------------------------
// 既存の findExtrema を，Brent法を使う形にアップグレード
// ---------------------------------------------------------

// 誤差関数をもとに極値点を探索する（グリッドサーチ + Brent法による精密化）．
func findExtrema(eFunc func(float64) float64, n int, gridSize int, a, b, brentTol float64) []float64 {
	fmt.Println("ステップ: 極値点探索 (Grid Search + Brent's Method)")
	var extrema []float64

	h := (b - a) / float64(gridSize)

	// グリッド上の値を事前計算（あるいはループ内で計算）して，
	// 「山（極大）」や「谷（極小）」になっている区間を見つける
	y_prev := eFunc(a)
	y_curr := eFunc(a + h)

	// 端点の処理: 端点が極値の可能性があるかチェック
	// (簡易的に，端点も候補に入れておき，後でソートして選抜する戦略をとる)
	extrema = append(extrema, a)

	for i := 1; i < gridSize-1; i++ {
		x := a + float64(i)*h
		y_next := eFunc(x + h)

		// 探索範囲の定義 (グリッドの前後)
		searchMin := x - h
		searchMax := x + h

		// 極大値の候補発見 (山)
		if y_curr > y_prev && y_curr > y_next {
			// 極大値を探す = -eFunc を最小化する
			negFunc := func(t float64) float64 { return -eFunc(t) }
			refinedX := brentMinimize(negFunc, searchMin, searchMax, brentTol, len(extrema)+1)
			extrema = append(extrema, refinedX)
		}

		// 極小値の候補発見 (谷)
		if y_curr < y_prev && y_curr < y_next {
			// 極小値を探す = eFunc を最小化する
			refinedX := brentMinimize(eFunc, searchMin, searchMax, brentTol, len(extrema)+1)
			extrema = append(extrema, refinedX)
		}

		y_prev = y_curr
		y_curr = y_next
	}

	extrema = append(extrema, b)

	// --- 以下は元のロジックと同じ選抜処理 ---

	// 絶対値が大きい順にソートして，上位の候補を残す
	// 重複除去（Brent法で同じ点に収束した場合の対策）も兼ねて近傍点をフィルタリングしても良いが
	// ここでは単純に絶対値ソートを行う
	sort.Slice(extrema, func(i, j int) bool {
		return math.Abs(eFunc(extrema[i])) > math.Abs(eFunc(extrema[j]))
	})

	// 必要な数だけ確保
	// 重複を避けるための簡易フィルタ（近すぎる点は除外）を入れるとより堅牢になります
	uniqueExtrema := make([]float64, 0, len(extrema))
	if len(extrema) > 0 {
		uniqueExtrema = append(uniqueExtrema, extrema[0])
		for i := 1; i < len(extrema); i++ {
			// 既に登録された点と十分離れていれば採用 (1e-6 は適当な閾値)
			isUnique := true
			for _, v := range uniqueExtrema {
				if math.Abs(v-extrema[i]) < 1e-6 {
					isUnique = false
					break
				}
			}
			if isUnique {
				uniqueExtrema = append(uniqueExtrema, extrema[i])
			}
		}
	}
	extrema = uniqueExtrema

	if len(extrema) < n+2 {
		log.Printf("警告: %d個の極値しか見つかりませんでした．期待値: %d個", len(extrema), n+2)
		// 足りない分を等間隔で補完
		for len(extrema) < n+2 {
			// ランダムあるいは等間隔で埋める（既存ロジック維持）
			x := a + float64(len(extrema))*(b-a)/float64(n+1)
			extrema = append(extrema, x)
		}
	}

	// 上位 n+2 個を採用してソート
	finalExtrema := extrema[:n+2]
	sort.Float64s(finalExtrema)
	return finalExtrema
}

// ---------------------------------------------------------
// 新規追加: Brent法による最小化アルゴリズム
// ---------------------------------------------------------

// brentMinimize は，区間 [a, b] における関数 f(x) の局所最小値を与える x を返します．
// Richard Brent "Algorithms for Minimization without Derivatives" に基づく実装です．
func brentMinimize(f func(float64) float64, a, b, tol float64, id int) float64 {
	// 黄金比
	const c = 0.3819660112501051 // (3 - sqrt(5)) / 2

	x := a + c*(b-a)
	w := x
	v := x
	e := 0.0 // 前回のステップ幅（放物線補間用）

	fx := f(x)
	fw := fx
	fv := fx

	d := 0.0 // 今回のステップ幅

	fmt.Printf("  >> [極値#%d] Brent探索開始: 初期区間[%.9f, %.9f]\n", id, a, b)

	for iter := 0; iter < 100; iter++ { // 安全のため最大反復回数を設定
		mid := (a + b) / 2.0
		tol1 := tol*math.Abs(x) + tol // 相対誤差 + 絶対誤差
		tol2 := 2.0 * tol1

		// 収束判定
		if math.Abs(x-mid) <= (tol2 - 0.5*(b-a)) {
			fmt.Printf("  << [極値#%d] Brent収束: iter=%d, x=%.12f, fx=%.5e\n", id, iter, x, fx)
			return x
		}

		method := "Golden"

		if math.Abs(e) > tol1 {
			// --- 放物線補間の試行 (SPI) ---
			r := (x - w) * (fx - fv)
			q := (x - v) * (fx - fw)
			p := (x-v)*q - (x-w)*r
			q = 2.0 * (q - r)
			if q > 0 {
				p = -p
			}
			q = math.Abs(q)

			etemp := e
			e = d

			// 放物線補間の受け入れ条件チェック
			// 1. |p| < |0.5 * q * etemp| : 前回のステップの半分未満か（収束速度の保証）
			// 2. a < x + p/q < b : 区間内にあるか
			if math.Abs(p) >= math.Abs(0.5*q*etemp) || p <= q*(a-x) || p >= q*(b-x) {
				// 失敗 -> 黄金分割へ
				if x >= mid {
					e = a - x
				} else {
					e = b - x
				}
				d = c * e
				method = "Golden (SPI reject)"
			} else {
				// 成功 -> 放物線補間ステップ採用
				d = p / q
				u := x + d
				if u-a < tol2 || b-u < tol2 {
					d = math.Copysign(tol1, mid-x)
				}
				method = "SPI"
			}
		} else {
			// --- 黄金分割探索 (Golden Section Search) ---
			if x >= mid {
				e = a - x
			} else {
				e = b - x
			}
			d = c * e
			method = "Golden"
		}

		fmt.Printf("     [#%d Iter %2d] %-19s x=%.12f, fx=%.5e, 区間=[%.9f, %.9f]\n", id, iter+1, method, x, fx, a, b)

		// 新しい点 u の決定（最小ステップ幅 tol1 を守る）
		u := 0.0
		if math.Abs(d) >= tol1 {
			u = x + d
		} else {
			u = x + math.Copysign(tol1, d)
		}

		fu := f(u)

		// 点の更新
		if fu <= fx {
			if u >= x {
				a = x
			} else {
				b = x
			}
			v = w
			fv = fw
			w = x
			fw = fx
			x = u
			fx = fu
		} else {
			if u < x {
				a = u
			} else {
				b = u
			}
			if fu <= fw || w == x {
				v = w
				fv = fw
				w = u
				fw = fu
			} else if fu <= fv || v == x || v == w {
				v = u
				fv = fu
			}
		}
	}
	return x
}

// 収束条件を判定する．
// checkConvergence: 最大誤差の前回との差分 |maxErr - prevMaxErr| が許容値未満なら収束とみなす．
// 返り値: (収束判定結果, 今回最大誤差)
func checkConvergence(prevMaxErr float64, eFunc func(float64) float64, extrema []float64, tolerance float64) (bool, float64) {
	fmt.Println("ステップ: 収束判定（最大誤差差分方式）")
	maxErr := 0.0
	for _, x := range extrema {
		err := math.Abs(eFunc(x))
		if err > maxErr {
			maxErr = err
		}
	}
	if math.IsInf(prevMaxErr, 1) { // 初回は比較不可
		fmt.Printf("今回最大誤差: %.9f (初回のため差分評価なし)\n", maxErr)
		return false, maxErr
	}
	diff := math.Abs(maxErr - prevMaxErr)
	fmt.Printf("今回最大誤差: %.9f, 前回最大誤差: %.9f, 差分: %.9e (許容値: %.1e)\n", maxErr, prevMaxErr, diff, tolerance)
	return diff < tolerance, maxErr
}

// formatFloatSlice: スライスを指定桁の精度で見やすく出力
func formatFloatSlice(xs []float64, prec int) string {
	if len(xs) == 0 {
		return "[]"
	}
	f := fmt.Sprintf("%%.%df", prec)
	parts := make([]string, len(xs))
	for i, v := range xs {
		parts[i] = fmt.Sprintf(f, v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// formatPolynomialLatex: 係数列から LaTeX 形式の多項式文字列を生成する．
// name: 多項式名 (例: P_{3,1}) / variable: 変数記号 / precision: 小数桁
// zeroThresh: これ未満の係数は 0 とみなしスキップ / simplifyUnit: 係数が ±1 の場合に係数を省略
func formatPolynomialLatex(coeffs []float64, name, variable string, precision int, zeroThresh float64, simplifyUnit bool) string {
	if len(coeffs) == 0 {
		return name + "(" + variable + ") = 0"
	}
	// 係数 -> 項へ
	terms := make([]string, 0, len(coeffs))
	format := fmt.Sprintf("%%.%df", precision)
	for power, c := range coeffs {
		if math.Abs(c) < zeroThresh { // ほぼ0は無視
			continue
		}
		sign := "+"
		if c < 0 {
			sign = "-"
		}
		absC := math.Abs(c)
		// 係数文字列
		coefStr := fmt.Sprintf(format, absC)
		// ±1 簡略化
		if simplifyUnit && math.Abs(absC-1.0) < zeroThresh {
			coefStr = "" // 省略
		}
		var termBody string
		switch power {
		case 0:
			// 定数項
			if coefStr == "" { // 係数が1または-1
				coefStr = "1" // 定数項は省略せず1表示
			}
			termBody = coefStr
		case 1:
			if coefStr == "" { // 係数±1
				termBody = variable
			} else {
				termBody = coefStr + variable
			}
		default:
			if coefStr == "" {
				termBody = variable + fmt.Sprintf("^{%d}", power)
			} else {
				termBody = coefStr + variable + fmt.Sprintf("^{%d}", power)
			}
		}
		// 最初の項は符号を先頭に付けない（負なら - を付ける）
		if len(terms) == 0 {
			if sign == "-" {
				terms = append(terms, "-"+termBody)
			} else {
				terms = append(terms, termBody)
			}
		} else {
			terms = append(terms, " "+sign+" "+termBody)
		}
	}
	if len(terms) == 0 { // 全て閾値未満
		return name + "(" + variable + ") = 0"
	}
	return name + "(" + variable + ") = " + strings.Join(terms, "")
}
