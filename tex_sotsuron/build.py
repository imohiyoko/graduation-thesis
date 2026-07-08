import re
import os

# ==========================================
# 設定
# ==========================================
# 親ファイル名
MAIN_TEX_FILE = "_imohiyoko_sotsuron.tex"

# 出力したいファイル名
# ここを希望のパス/ファイル名に変更してください
OUTPUT_FILE = "_imohiyoko_sotsuron_build.tex.txt"

# ==========================================
# スクリプト本体
# ==========================================

def resolve_file_path(base_path, relative_path):
    """
    相対パスを解決し，拡張子がない場合は.texを付与する
    """
    # 拡張子がなければ .tex をつける
    if not relative_path.lower().endswith('.tex'):
        relative_path += '.tex'
    
    # パスを結合
    full_path = os.path.join(os.path.dirname(base_path), relative_path)
    return full_path

def process_file(filepath):
    """
    ファイルを読み込み，\inputや\includeを展開して返す（再帰処理）
    """
    if not os.path.exists(filepath):
        print(f"Warning: File not found: {filepath}")
        return [f"% ERROR: File not found ({filepath})\n"]

    print(f"Processing: {filepath}")
    
    # \input{...} または \include{...} を探す正規表現
    # 行の先頭付近にあるもののみ対象（コメントアウトされている行は除外したいが，簡易的に行頭チェック）
    # グループ1にファイル名が入る
    input_pattern = re.compile(r'^\s*\\(?:input|include)\{([^}]+)\}')

    processed_lines = []

    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            lines = f.readlines()

        for line in lines:
            # コメント行（%から始まる）は展開処理をスキップしてそのまま出力
            if line.strip().startswith('%'):
                processed_lines.append(line)
                continue

            match = input_pattern.match(line)
            if match:
                # 読み込むファイル名を取得
                target_filename = match.group(1)
                target_filepath = resolve_file_path(filepath, target_filename)

                # 結合時の区切りがわかるようにコメントを追加
                processed_lines.append(f"\n% =========================================================\n")
                processed_lines.append(f"% START OF FILE: {target_filename}\n")
                processed_lines.append(f"% =========================================================\n")
                
                # 再帰的にファイルを読み込む
                processed_lines.extend(process_file(target_filepath))
                
                processed_lines.append(f"\n% =========================================================\n")
                processed_lines.append(f"% END OF FILE: {target_filename}\n")
                processed_lines.append(f"% =========================================================\n")
            else:
                processed_lines.append(line)

    except Exception as e:
        print(f"Error reading {filepath}: {e}")
        return [f"% ERROR reading {filepath}\n"]

    return processed_lines

def main():
    if not os.path.exists(MAIN_TEX_FILE):
        print(f"Error: Main file '{MAIN_TEX_FILE}' not found.")
        return

    print("Starting TeX combination...")
    final_lines = process_file(MAIN_TEX_FILE)

    try:
        with open(OUTPUT_FILE, 'w', encoding='utf-8') as f:
            f.writelines(final_lines)
        print("-" * 30)
        print(f"Successfully created: {OUTPUT_FILE}")
        print("-" * 30)
    except Exception as e:
        print(f"Error writing output file: {e}")

if __name__ == "__main__":
    main()