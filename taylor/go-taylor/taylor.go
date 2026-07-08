package main

import (
	"fmt"
	"math"
)

type DerivativeFunc func(n int, x float64) float64

func main() {
	runInteractiveSession()
}

func runInteractiveSession() {
	rangeStart, rangeEnd := -3.14, 3.14
	maxDegree := 30
	threshold := 1e-4

	for {
		// 現在の設定状況を表示してあげる（親切設計）
		center := (rangeStart + rangeEnd) / 2.0
		fmt.Println("\n========================================")
		fmt.Printf("  現在の設定: 区間[%.2f, %.2f], 中心 %.2f\n", rangeStart, rangeEnd, center)
		fmt.Printf("             許容誤差 %.1e, 最大次数 %d\n", threshold, maxDegree)
		fmt.Println("========================================")

		fmt.Println("操作を選んでください:")
		fmt.Println("1: exp(x) の収束判定")
		fmt.Println("2: sin(x) の収束判定")
		fmt.Println("3: cos(x) の収束判定")
		fmt.Println("9: 設定を変更する") // ★改善3: 設定変更メニューを追加
		fmt.Println("0: 終了")
		fmt.Print(">> 選択: ")

		var choice int
		_, err := fmt.Scan(&choice)
		if err != nil {
			fmt.Println("数値で入力してください．")
			var discard string
			fmt.Scanln(&discard)
			continue
		}

		if choice == 0 {
			fmt.Println("終了します．")
			break
		}

		switch choice {
		case 1:
			// 今の設定値を使って即実行！
			analyzeConvergence("exp(x)", center, maxDegree, threshold, rangeStart, rangeEnd, derivExp, math.Exp)
		case 2:
			analyzeConvergence("sin(x)", center, maxDegree, threshold, rangeStart, rangeEnd, derivSin, math.Sin)
		case 3:
			analyzeConvergence("cos(x)", center, maxDegree, threshold, rangeStart, rangeEnd, derivCos, math.Cos)

		case 9:
			// ★設定変更モード
			fmt.Println("\n--- 設定の変更 ---")
			fmt.Printf("誤差評価区間の開始 (現在: %.2f): ", rangeStart)
			fmt.Scan(&rangeStart)
			fmt.Printf("誤差評価区間の終了 (現在: %.2f): ", rangeEnd)
			fmt.Scan(&rangeEnd)
			fmt.Printf("許容誤差（閾値） (現在: %.1e): ", threshold)
			fmt.Scan(&threshold)
			fmt.Printf("最大探索次数 (現在: %d): ", maxDegree)
			fmt.Scan(&maxDegree)
			fmt.Println("設定を更新しました．")

		default:
			fmt.Println("無効な選択です．")
		}
	}
}

// （以下の関数は前回と同じなので省略なしで動くよう，そのまま記載します）

func analyzeConvergence(name string, center float64, maxLimit int, threshold, start, end float64, dFunc DerivativeFunc, trueFunc func(float64) float64) {
	fmt.Printf("\n--- %s の収束次数探索 ---\n", name)
	converged := false

	for n := 1; n <= maxLimit; n++ {
		maxErr, _ := calcMaxError(center, n, start, end, dFunc, trueFunc)
		fmt.Printf("次数 %2d | 最大誤差: %.4e", n, maxErr)

		if maxErr < threshold {
			fmt.Println(" -> 収束！")
			converged = true
			break
		}
		fmt.Println()
	}

	if !converged {
		fmt.Println("\n⚠ 指定次数内では収束しませんでした．")
	}
}

func calcMaxError(center float64, degree int, start, end float64, dFunc DerivativeFunc, trueFunc func(float64) float64) (float64, float64) {
	mesh := 1000
	step := (end - start) / float64(mesh)
	maxErr := 0.0
	worstX := start

	for i := 0; i <= mesh; i++ {
		x := start + float64(i)*step
		approx := calculateTaylor(x, center, degree, dFunc)
		trueVal := trueFunc(x)
		diff := math.Abs(approx - trueVal)
		if diff > maxErr {
			maxErr = diff
			worstX = x
		}
	}
	return maxErr, worstX
}

func calculateTaylor(x float64, center float64, degree int, dFunc DerivativeFunc) float64 {
	sum := 0.0
	powerFactor := 1.0
	for i := 0; i <= degree; i++ {
		deriv := dFunc(i, center)
		sum += deriv * powerFactor
		powerFactor = powerFactor * (x - center) / float64(i+1)
	}
	return sum
}

func derivExp(n int, x float64) float64 { return math.Exp(x) }

func derivSin(n int, x float64) float64 {
	switch n % 4 {
	case 0:
		return math.Sin(x)
	case 1:
		return math.Cos(x)
	case 2:
		return -math.Sin(x)
	case 3:
		return -math.Cos(x)
	}
	return 0
}

func derivCos(n int, x float64) float64 {
	switch n % 4 {
	case 0:
		return math.Cos(x)
	case 1:
		return -math.Sin(x)
	case 2:
		return -math.Cos(x)
	case 3:
		return math.Sin(x)
	}
	return 0
}
