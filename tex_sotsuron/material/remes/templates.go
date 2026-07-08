package main

import (
	"fmt"
	"strings"
)

// ==========================================
// TeXテンプレート定義
// ==========================================

// Grid法用のテンプレート（3章用）
const TexTemplateGrid = `%% 3章用 (Grid)

前述の理論に基づき，本研究ではGo言語を用いてRemesアルゴリズムを実装した．
以下の図表は，関数 $f(x) = %FUNC_TEX%$ ($n=%DEGREE%$) を近似した際の収束過程を示したものである．

表\ref{tab:remes-grid-log}は，各反復で算出された参照均等誤差 $|E|$ と，その時点での誤差関数の極値の絶対値の最大値と最小値の差（Deviation）の推移を示したものである．

本実装における収束判定条件は，すべてのパラメータの変化量が機械イプシロン（$\epsilon_{\text{mach}} \approx 2.22 \times 10^{-16}$）以下になった時点とする．
\begin{itemize}
  \item 近似多項式の係数ベクトル $\boldsymbol{c}$ の最大変化量：
  \[ \|\boldsymbol{c}^{(k+1)} - \boldsymbol{c}^{(k)}\|_\infty \le \epsilon_{\text{mach}}. \]
  \item 参照点ベクトル $\boldsymbol{\xi}$ の最大変化量：
  \[ \|\boldsymbol{\xi}^{(k+1)} - \boldsymbol{\xi}^{(k)}\|_\infty \le \epsilon_{\text{mach}}. \]
  \item 均等誤差パラメータ $E$ の変化量：
  \[ |E^{(k+1)} - E^{(k)}| \le \epsilon_{\text{mach}}. \]
\end{itemize}
これにより，数値計算上の限界精度まで解が収束したことを保証する．

表中の Deviation は，係数が収束に向かうにつれて交替定理の条件（等振幅性）が満たされていく様子を表す指標であり，理論通り急速に 0 に収束していることが確認できる．

図\ref{fig:remes-grid-convergence}は，この Deviation の推移を可視化したものである．
初期段階では不揃いであった誤差の波形が，最終的には振幅が一定の美しい等振動波形へと収束していることが見て取れる．

`

// Brent法用のテンプレート（5章用）
const TexTemplateBrent = `%% 5章用 (Brent)

以下の図表は，Brent法による極値探索を用いて，関数 $f(x) = %FUNC_TEX%$ ($n=%DEGREE%$) を近似した際の収束過程を示したものである．

表\ref{tab:remes-brent-log}は，各反復で算出された参照均等誤差 $|E|$ と，Brent法で高精度に求められた実際の最大誤差，およびその差の推移である．
グリッド極値点探索と比較して，より高精度な収束（$10^{-15}$ オーダー）が達成されている．

図\ref{fig:remes-brent-convergence}は，反復に伴う誤差の収束過程を可視化したものである．
Brent法により，機械イプシロン精度に近い等振動性が実現されていることが確認できる．


`

// 比較用のテンプレート（6章用）
const TexTemplateComparison = `%% 6章用 (Comparison)

以下の図表は，グリッド極値点探索法とBrent法の性能を比較したものである（対象関数: $f(x) = %FUNC_TEX%$，次数: $n=%DEGREE%$）．

表\ref{tab:remes-comparison-summary}は，両手法の収束性能，計算時間，最終精度を比較した総合評価である．
Brent法は計算時間がやや増加するものの，極値探索精度の向上により，より高精度な最良近似多項式が得られることが示されている．

図\ref{fig:remes-comparison-deviation}は，各極値点における理論値からの偏差を可視化したものである．
グリッド極値点探索では離散化誤差により微小なズレが残るのに対し，Brent法では機械イプシロンレベルの精度が達成されている．

`

// テンプレート変数を置換する関数
func fillTemplate(template string, funcTexName string, degree int) string {
	replacer := NewTexReplacer(funcTexName, degree)
	return replacer.Replace(template)
}

// TexReplacer: テンプレート変数を置換するヘルパー
func NewTexReplacer(funcTexName string, degree int) *strings.Replacer {
	return strings.NewReplacer(
		"%FUNC_TEX%", funcTexName,
		"%DEGREE%", fmt.Sprintf("%d", degree),
	)
}
