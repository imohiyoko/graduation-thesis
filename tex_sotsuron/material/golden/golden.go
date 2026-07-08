package main

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

// --- 設定パラメータ ---
const (
	TotalTrials = 1_000_000 // 試行回数 (100万回)
	SigmaRange  = 5.0       // 探索範囲 (±5σ)
	Tolerance   = 1e-6      // 収束条件 (許容誤差)
)

// --- データ構造 ---

// TrialResult : 1回の試行結果
type TrialResult struct {
	GSS_Calls     int     // 黄金分割法の関数呼び出し回数
	Ternary_Calls int     // 三分探索の関数呼び出し回数
	Random_Error  float64 // ランダム探索の誤差(比較用)
	GSS_Error     float64 // 黄金分割法の誤差
	Ternary_Error float64 // 三分探索の誤差
}

// --- 数学関数 ---

// CalcNormalDist : 正規分布 (呼び出し回数カウント付き)
// 実際の重い処理をシミュレートするため、counterポインタを受け取ります
func CalcNormalDist(x, mu, sigma float64, counter *int) float64 {
	*counter++ // 呼び出し回数をインクリメント
	return math.Exp(-math.Pow(x-mu, 2) / (2 * math.Pow(sigma, 2)))
}

// GoldenSectionSearch : 黄金分割法
// 特徴: 前回の計算結果を再利用するため、ループごとの計算は1回で済む
func GoldenSectionSearch(mu, sigma, a, b, tol float64) (float64, int, int) {
	calls := 0
	gr := (math.Sqrt(5) - 1) / 2 // 黄金比の逆数 (約0.618)

	c := b - gr*(b-a)
	d := a + gr*(b-a)

	// 初回のみ2回計算
	fc := CalcNormalDist(c, mu, sigma, &calls)
	fd := CalcNormalDist(d, mu, sigma, &calls)

	iter := 0
	for (b - a) > tol {
		iter++
		if fc > fd {
			// 最大値は左側にある -> 右端 b を d に狭める
			b = d
			d = c   // d は計算済み(c)を再利用！
			fd = fc // 値も再利用！

			// 新しい c だけ計算
			c = b - gr*(b-a)
			fc = CalcNormalDist(c, mu, sigma, &calls)
		} else {
			// 最大値は右側にある -> 左端 a を c に狭める
			a = c
			c = d   // c は計算済み(d)を再利用！
			fc = fd // 値も再利用！

			// 新しい d だけ計算
			d = a + gr*(b-a)
			fd = CalcNormalDist(d, mu, sigma, &calls)
		}
	}
	return (a + b) / 2, iter, calls
}

// TernarySearch : 三分探索 (二分法的アプローチ)
// 特徴: 区間を3等分して真ん中の2点を比較する．毎回2回計算が必要．
func TernarySearch(mu, sigma, a, b, tol float64) (float64, int, int) {
	calls := 0
	iter := 0

	for (b - a) > tol {
		iter++
		// 区間を3等分する2点
		m1 := a + (b-a)/3
		m2 := b - (b-a)/3

		// 毎回2点とも計算しなおす必要がある
		f1 := CalcNormalDist(m1, mu, sigma, &calls)
		f2 := CalcNormalDist(m2, mu, sigma, &calls)

		if f1 < f2 {
			a = m1 // 右側を残す
		} else {
			b = m2 // 左側を残す
		}
	}
	return (a + b) / 2, iter, calls
}

// RandomSearch : ランダム探索 (ベースライン)
func RandomSearch(mu, sigma, a, b float64, limit int, rng *rand.Rand) float64 {
	bestX := a
	bestY := -1.0
	calls := 0 // ダミー用

	for i := 0; i < limit; i++ {
		x := a + rng.Float64()*(b-a)
		y := CalcNormalDist(x, mu, sigma, &calls)
		if y > bestY {
			bestY = y
			bestX = x
		}
	}
	return bestX
}

// --- ワーカー ---

func worker(jobs <-chan int, results chan<- TrialResult, wg *sync.WaitGroup) {
	defer wg.Done()
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	for range jobs {
		// ランダムな正規分布を生成
		targetMu := (rng.Float64() * 200) - 100
		targetSigma := 0.5 + rng.Float64()*4.5

		startA := targetMu - (targetSigma * SigmaRange)
		startB := targetMu + (targetSigma * SigmaRange)

		// 1. 黄金分割法
		gssX, _, gssCalls := GoldenSectionSearch(targetMu, targetSigma, startA, startB, Tolerance)

		// 2. 三分探索 (同じ許容誤差になるまで実行)
		tsX, _, tsCalls := TernarySearch(targetMu, targetSigma, startA, startB, Tolerance)

		// 3. ランダム探索 (GSSと同じ回数だけ計算させる)
		randX := RandomSearch(targetMu, targetSigma, startA, startB, gssCalls, rng)

		results <- TrialResult{
			GSS_Calls:     gssCalls,
			Ternary_Calls: tsCalls,
			GSS_Error:     math.Abs(gssX - targetMu),
			Ternary_Error: math.Abs(tsX - targetMu),
			Random_Error:  math.Abs(randX - targetMu),
		}
	}
}

// --- メイン処理 ---

func main() {
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)

	fmt.Printf("=== アルゴリズム性能比較検証 ===\n")
	fmt.Printf("総試行回数: %d回\n", TotalTrials)
	fmt.Printf("目標精度(許容誤差): %e\n", Tolerance)
	fmt.Println("--------------------------------------------------")

	jobs := make(chan int, TotalTrials)
	results := make(chan TrialResult, TotalTrials)
	var wg sync.WaitGroup

	startTime := time.Now()

	for w := 0; w < numCPU; w++ {
		wg.Add(1)
		go worker(jobs, results, &wg)
	}

	go func() {
		for i := 0; i < TotalTrials; i++ {
			jobs <- i
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	// 集計用変数
	var sumGSSCalls, sumTSCalls int64
	var sumGSSErr, sumTSErr, sumRandErr float64

	processed := 0
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	fmt.Println("計算中...")

	for res := range results {
		sumGSSCalls += int64(res.GSS_Calls)
		sumTSCalls += int64(res.Ternary_Calls)
		sumGSSErr += res.GSS_Error
		sumTSErr += res.Ternary_Error
		sumRandErr += res.Random_Error
		processed++

		if processed%(TotalTrials/10) == 0 {
			fmt.Printf("進捗: %d%%\n", processed*100/TotalTrials)
		}
	}

	elapsed := time.Since(startTime)

	avgGSSCalls := float64(sumGSSCalls) / float64(TotalTrials)
	avgTSCalls := float64(sumTSCalls) / float64(TotalTrials)

	fmt.Println("\n=== 検証結果: 同じ精度に到達するのに必要なコスト ===")
	fmt.Printf("処理時間: %s\n", elapsed)
	fmt.Println("--------------------------------------------------")
	fmt.Printf("アルゴリズム     | 平均関数呼び出し回数 | 到達精度(誤差)\n")
	fmt.Println("--------------------------------------------------")
	fmt.Printf("黄金分割法(GSS)  | %15.2f回 | %.2e\n", avgGSSCalls, sumGSSErr/float64(TotalTrials))
	fmt.Printf("三分探索(Ternary)| %15.2f回 | %.2e\n", avgTSCalls, sumTSErr/float64(TotalTrials))
	fmt.Printf("ランダム探索     | (GSSと同じ回数)  | %.2e (桁違いに悪い)\n", sumRandErr/float64(TotalTrials))
	fmt.Println("--------------------------------------------------")

	efficiency := (avgTSCalls - avgGSSCalls) / avgTSCalls * 100
	fmt.Printf("\n【結論】\n")
	fmt.Printf("黄金分割法は、三分探索と比較して計算コストを約 %.1f%% 削減しました．\n", efficiency)
	fmt.Println("理由: 三分探索は各ステップで「2回」関数計算を行いますが、")
	fmt.Println("      黄金分割法は幾何学的な性質により「1回」の計算で済みます．")
	fmt.Println("      重い計算処理を最適化する場合、この約1.5倍〜2倍の効率差は巨大なコスト削減になります．")
}
