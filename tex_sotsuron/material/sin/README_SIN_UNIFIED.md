# sin関数の統合シミュレーション

## 概要
このプログラムは、sin関数に対して以下の2つの近似手法を実行し、結果を可視化します：
1. **テイラー展開**（Taylor expansion）
2. **等間隔補間**（Lagrange interpolation with equidistant nodes）

## 主な特徴

### 統一された設定
- **範囲**: `[-π, π]` に統一
- **最大次数**: 25次まで計算
- **出力次数**: `[1, 5, 11]` （カスタマイズ可能）

### ラベル命名規則
生成されるTeXファイルのfigureラベルは以下の規則に従います：

#### テイラー展開
- 最初の図: `\label{fig:sin-taylor-first}`
- 中間の図: `\label{fig:sin-taylor-2}`, `\label{fig:sin-taylor-3}`, ...
- 最後の図: `\label{fig:sin-taylor-last}`

#### 等間隔補間
- 最初の図: `\label{fig:sin-interp-first}`
- 中間の図: `\label{fig:sin-interp-2}`, `\label{fig:sin-interp-3}`, ...
- 最後の図: `\label{fig:sin-interp-last}`

### TeX図の構造
各figureは以下の構造になっています：
```latex
\begin{figure}[H]
  \centering
  \begin{minipage}[b]{0.48\linewidth}
    \centering
    \includegraphics[width=1.0\linewidth]{material/sin/sin_taylor_n01.png}
  \end{minipage}
  \hfill
  \begin{minipage}[b]{0.48\linewidth}
    \centering
    \includegraphics[width=1.0\linewidth]{material/sin/sin_taylor_err_n01.png}
  \end{minipage}
  \caption{$f(x) = \sin(x)$のテイラー展開（3次多項式）による近似結果（左）と誤差（右）}
  \label{fig:sin-taylor-first}
\end{figure}
```

**重要**: 各minipage内には個別のcaptionがなく、figure全体で1つのcaptionのみを持ちます．

## 設定のカスタマイズ

### 出力次数の変更
```go
var OutputDegrees = []int{1, 5, 11} // この配列を編集
```

例：
- `[]int{1, 3, 5, 7}` - 次数1, 3, 5, 7を出力
- `[]int{5, 10, 15, 20}` - 次数5, 10, 15, 20を出力

### 範囲の変更
```go
const (
	RangeStart = -math.Pi // 開始点
	RangeEnd   = math.Pi  // 終了点
)
```

### テイラー展開の中心点
```go
const CenterPoint = 0.0 // x=0で展開
```

## 生成されるファイル

### テイラー展開
- **グラフ**: `sin_taylor_n01.png` ~ `sin_taylor_n25.png`
- **誤差**: `sin_taylor_err_n01.png` ~ `sin_taylor_err_n25.png`
- **TeX**: `sin_taylor_result.tex`

### 等間隔補間
- **グラフ**: `sin_interp_n01.png` ~ `sin_interp_n25.png`
- **誤差**: `sin_interp_err_n01.png` ~ `sin_interp_err_n25.png`
- **TeX**: `sin_interp_result.tex`

## 実行方法

```bash
go run sin_complete_unified.go
```

## 出力例

実行すると以下のようなメッセージが表示されます：
```
=== sin関数のテイラー展開と補間の統合シミュレーション ===

--- テイラー展開を開始します（中心点: 0.000000）---
N=01: Saved sin_taylor_n01.png and sin_taylor_err_n01.png
N=02: Saved sin_taylor_n02.png and sin_taylor_err_n02.png
...
sin_taylor_result.tex の生成完了

--- 等間隔補間を開始します ---
N=01: Saved sin_interp_n01.png and sin_interp_err_n01.png
N=02: Saved sin_interp_n02.png and sin_interp_err_n02.png
...
sin_interp_result.tex の生成完了

=== シミュレーション完了 ===
```

## 旧ファイルとの違い

### sinTaylor (旧)
- 範囲: `[-2π, 2π]`
- 出力次数: `[1, 3, 5, 7, 9, 11]`
- minipage内に個別caption

### sin (旧)
- 範囲: `[-π, π]`
- 出力次数: `[3, 7, 15]`
- minipage内に個別caption

### sin_complete_unified.go (新)
- 範囲: `[-π, π]` に統一
- 出力次数: `[1, 5, 11]` に統一（カスタマイズ可能）
- figure全体で1つのcaption
- first/last/番号のラベル規則

## メンテナンス

新しい近似手法を追加する場合：
1. 近似関数を実装
2. `runXxxSimulation()` 関数を追加
3. `main()` 関数で呼び出し
4. ラベル命名規則に従ってTeX生成部分を実装
