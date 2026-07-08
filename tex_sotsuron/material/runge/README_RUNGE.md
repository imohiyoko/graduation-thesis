# Runge 現象シミュレーション

## 概要

`runge_unified.go` は、ラグランジュ補間におけるルンゲ現象を可視化するための統一されたプログラムです．

以下の2つのモードで動作できます：
- **等間隔補間** (`--mode equidistant`) : ルンゲ現象が顕著に現れる場合
- **チェビシェフ補間** (`--mode chebyshev`) : ルンゲ現象を回避する場合

**重要**: `runge_unified.go` は `material/runge/` に一元管理されています．

## 使用方法

### 等間隔補間（ルンゲ現象の例）

```bash
cd material/runge
go run runge_unified.go --mode equidistant --max-n 25
```

出力ファイル：
- `runge_n01.png` ～ `runge_n25.png` : 補間多項式の可視化
- `runge_err_n01.png` ～ `runge_err_n25.png` : 誤差関数の可視化
- `runge_result.tex` : TeX 統合ファイル（自動生成）

### チェビシェフ補間（改善例）

```bash
cd material/runge_Chebyshev
go run ../runge/runge_unified.go --mode chebyshev --max-n 25
```

出力ファイル：
- `runge_Chebyshev_n01.png` ～ `runge_Chebyshev_n25.png` : 補間多項式の可視化
- `runge_err_Chebyshev_n01.png` ～ `runge_err_Chebyshev_n25.png` : 誤差関数の可視化
- `runge_Chebyshev_result.tex` : TeX 統合ファイル（自動生成）

### 一括生成スクリプト

```bash
cd material/runge
bash generate_runge_plots.sh
```

このスクリプトは両モードで自動実行し、すべてのファイルを生成します．

## コマンドラインオプション

| オプション | デフォルト | 説明 |
|-----------|----------|------|
| `--mode` | `equidistant` | 補間モード: `equidistant` または `chebyshev` |
| `--max-n` | `25` | 最大多項式次数 |

## 出力ファイル

### PNG ファイル

各実行で、1～25 次（またはカスタム最大次数）のグラフが生成されます．

#### 補間グラフ（`*_nNN.png`）
- 黒線：元の関数 $f(x) = \frac{1}{1 + 25x^2}$
- 破線（等間隔の場合は赤、チェビシェフの場合は青）：補間多項式 $P_n(x)$

#### 誤差グラフ（`*_err_nNN.png`）
- 誤差関数 $e(x) = P_n(x) - f(x)$ のプロット
- 最大絶対誤差値を凡例に表示

### TeX ファイル

`*_result.tex` は以下の特定の次数（$n=4, 7, 15$）のみを含みます：
```tex
\begin{figure}[H]
  \centering
  \includegraphics[width=0.7\linewidth]{...}
  \caption{...}
\end{figure}
```

01.tex で次のように参照します：
```tex
\input{material/runge/runge_result}
\input{material/runge_Chebyshev/runge_Chebyshev_result}
```

## ファイル構成

```
material/
├── runge/
│   ├── runge_unified.go         ← メインプログラム（両モードに対応）
│   ├── generate_runge_plots.sh   ← 一括生成スクリプト
│   ├── README_RUNGE.md           ← このファイル
│   ├── runge_n*.png
│   ├── runge_err_n*.png
│   └── runge_result.tex          （自動生成）
│
└── runge_Chebyshev/
    ├── runge_n*.png
    ├── runge_err_n*.png
    └── runge_Chebyshev_result.tex （自動生成）
```

## 特徴

✅ **単一のプログラムで複数モード対応** : `material/runge/` に一元化
✅ **柔軟な次数指定** : `--max-n` で任意の次数までグラフ生成可能
✅ **自動 TeX 統合** : Go 実行時に `*_result.tex` が自動生成
✅ **ディレクトリ分離** : モードに応じて異なるディレクトリに出力
✅ **保守性向上** : コードは1つの場所で管理

## 開発履歴

- **v2.0**: `material/runge/` に完全統合
  - 両ディレクトリ間の重複を排除
  - `runge_unified.go` を1つの場所で管理
  
- **v1.0**: 統一 Go プログラムとして実装
  - `material/runge/runge.go` と `material/runge_Chebyshev/runge.go` を統合
  - コマンドラインフラグで動的に制御
  - TeX 生成機能を搭載

## 出力ファイル

### PNG ファイル

各実行で、1～25 次（またはカスタム最大次数）のグラフが生成されます．

#### 補間グラフ（`*_nNN.png`）
- 黒線：元の関数 $f(x) = \frac{1}{1 + 25x^2}$
- 破線（等間隔の場合は赤、チェビシェフの場合は青）：補間多項式 $P_n(x)$

#### 誤差グラフ（`*_err_nNN.png`）
- 誤差関数 $e(x) = P_n(x) - f(x)$ のプロット
- 最大絶対誤差値を凡例に表示

### TeX ファイル

`*_result.tex` は以下の特定の次数（$n=4, 7, 15$）のみを含みます：
```tex
\begin{figure}[H]
  \centering
  \includegraphics[width=0.7\linewidth]{...}
  \caption{...}
\end{figure}
```

01.tex で次のように参照します：
```tex
\input{material/runge/runge_result}
\input{material/runge_Chebyshev/runge_Chebyshev_result}
```

## 特徴

✅ **単一のプログラムで複数モード対応** : 重複したコードを排除
✅ **柔軟な次数指定** : `--max-n` で任意の次数までグラフ生成可能
✅ **自動 TeX 統合** : Go 実行時に `*_result.tex` が自動生成
✅ **ディレクトリ分離** : モードに応じて異なるディレクトリに出力

## 開発履歴

- **v1.0**: 統一 Go プログラムとして実装
  - `material/runge/runge.go` と `material/runge_Chebyshev/runge.go` を統合
  - コマンドラインフラグで動的に制御
  - TeX 生成機能を搭載
