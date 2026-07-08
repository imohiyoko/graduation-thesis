# Remes Algorithm - Template-based TeX Generation

## 概要

このプログラムは、Remesアルゴリズムを用いて最良近似多項式を計算し、結果をLaTeXファイルとして出力します．

**重要な特徴：**
- **テンプレート形式**で説明文を自動生成
- 関数と次数を変更しても、説明文が自動的に更新される
- 別ファイル不要、すべてGo内で完結

## ファイル構成

```
remes.go        - メインプログラム（アルゴリズム実装）
templates.go    - TeXテンプレート定義
```

## 使い方

### 1. デフォルト設定（ルンゲ関数、次数15）

そのまま実行：

```bash
go run *.go
```

生成されるファイル：
- `remes_grid_result.tex`
- `remes_brent_result.tex`
- `remes_comparison_result.tex`

### 2. カスタム設定（別の関数・次数）

`remes.go` の main 関数内で、コメントを切り替え：

```go
// デフォルトをコメントアウト
// globalConfig = NewDefaultRemesConfig()

// カスタム設定をコメントイン
globalConfig = NewRemesConfigWithCustom(
    func(x float64) float64 { return math.Exp(x) },  // 関数
    "e^x",                                            // TeX表現
    4,                                                 // 次数
    "exp",                                             // ファイル名プレフィックス
)
```

生成されるファイル：
- `exp_grid_result.tex`
- `exp_brent_result.tex`
- `exp_comparison_result.tex`

### 3. テンプレートのカスタマイズ

説明文を変更したい場合は、`templates.go` 内の定数を編集：

- `TexTemplateGrid` - Grid法用（3章）
- `TexTemplateBrent` - Brent法用（5章）
- `TexTemplateComparison` - 比較用（6章）

**使用可能な変数：**
- `%FUNC_TEX%` - 関数のTeX表現（例: `e^x`, `\frac{1}{1+25x^2}`）
- `%DEGREE%` - 多項式次数

## ワークフロー

```
[あなたがGoを実行]
    ↓
[templates.go内のテンプレート文字列を読み込み]
    ↓
[設定から関数名と次数を取得]
    ↓
[テンプレート変数を置換]
    ↓
[完成したTeXファイルを生成・保存]
    ↓
[後でLaTeXコンパイル時に\inputで読み込む]
```

## 関数のTeX表現例

| 関数 | Go実装 | TeX表現 |
|------|--------|---------|
| 指数関数 | `math.Exp(x)` | `e^x` |
| ルンゲ関数 | `1.0 / (1.0 + 25.0*x*x)` | `\frac{1}{1+25x^2}` |
| 正弦関数 | `math.Sin(x)` | `\sin(x)` |
| 多項式 | `x*x*x - 2*x + 1` | `x^3 - 2x + 1` |

## 利点

1. **一貫性**: 関数や次数を変更しても、説明文が自動更新される
2. **保守性**: テンプレートを一箇所で管理できる
3. **効率性**: 手動編集不要、プログラム実行だけで完結
4. **柔軟性**: 複数の関数・次数を簡単にテスト可能

## 注意事項

生成されたTeXファイルは**Goプログラム実行時に上書き**されます．
手動で編集しても、次回実行時に失われます．

説明文を変更したい場合は、必ず `templates.go` を編集してください．
