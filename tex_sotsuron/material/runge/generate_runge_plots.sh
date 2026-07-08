#!/bin/bash

# runge_unified.go を使った PNG と TeX ファイルの生成スクリプト
# material/runge/ に統合された runge_unified.go を使用

RUNGE_DIR="$(dirname "$0")"
RUNGE_CHEBYSHEV_DIR="$(dirname "$0")/../runge_Chebyshev"

echo "=== Runge 現象シミュレーション: 等間隔補間 ==="
cd "$RUNGE_DIR"

# 等間隔補間版の生成
echo "等間隔補間のグラフと TeX ファイルを生成中..."
go run runge_unified.go --mode equidistant --max-n 25

if [ -f runge_result.tex ]; then
    echo "✓ runge_result.tex が生成されました"
else
    echo "✗ runge_result.tex の生成に失敗しました"
fi

echo ""
echo "=== Runge 現象シミュレーション: チェビシェフ補間 ==="

# チェビシェフ補間版の生成（別ディレクトリで実行）
cd "$RUNGE_CHEBYSHEV_DIR"
echo "チェビシェフ補間のグラフと TeX ファイルを生成中..."
go run ../runge/runge_unified.go --mode chebyshev --max-n 25

if [ -f runge_Chebyshev_result.tex ]; then
    echo "✓ runge_Chebyshev_result.tex が生成されました"
else
    echo "✗ runge_Chebyshev_result.tex の生成に失敗しました"
fi

echo ""
echo "=== 生成完了 ==="
echo "生成ファイル:"
echo "  - material/runge/runge_n*.png (等間隔補間)"
echo "  - material/runge/runge_err_n*.png (等間隔補間の誤差)"
echo "  - material/runge/runge_result.tex"
echo "  - material/runge_Chebyshev/runge_Chebyshev_n*.png (チェビシェフ補間)"
echo "  - material/runge_Chebyshev/runge_err_Chebyshev_n*.png (チェビシェフ補間の誤差)"
echo "  - material/runge_Chebyshev/runge_Chebyshev_result.tex"
