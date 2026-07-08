import os
import shutil

def process_thesis_materials(source_dir):
    # パスの末尾に区切り文字がある場合は削除
    if source_dir.endswith(os.sep):
        source_dir = source_dir[:-1]
    
    # 親ディレクトリとフォルダ名を取得
    parent_dir = os.path.dirname(os.path.abspath(source_dir))
    base_name = os.path.basename(source_dir)
    
    # 出力先フォルダ名（元のフォルダ名 + _export）
    dest_dir = os.path.join(parent_dir, base_name + "_export")

    if os.path.exists(dest_dir):
        print(f"エラー: 出力先フォルダ '{dest_dir}' が既に存在します．\n削除するかリネームしてから再実行してください．")
        return

    print(f"処理対象: {source_dir}")
    print(f"出力先: {dest_dir}")
    print("-" * 30)

    # 再帰的に探索
    file_count = 0
    for root, dirs, files in os.walk(source_dir):
        # 出力先のディレクトリパスを作成
        rel_path = os.path.relpath(root, source_dir)
        target_root = os.path.join(dest_dir, rel_path)

        os.makedirs(target_root, exist_ok=True)

        for filename in files:
            src_file = os.path.join(root, filename)
            
            # .tex または .go ファイルの場合：中身をコピーして拡張子 .txt を付与
            if filename.endswith(('.tex', '.go')):
                dest_file = os.path.join(target_root, filename + ".txt")
                try:
                    # バイナリモードでコピー（エンコーディング問題を回避）
                    shutil.copy2(src_file, dest_file)
                    print(f"[COPY] {filename} -> {filename}.txt")
                    file_count += 1
                except Exception as e:
                    print(f"[ERROR] {filename} のコピーに失敗: {e}")
            
            # その他のファイルは無視

    print("-" * 30)
    print(f"完了しました．合計 {file_count} 個のファイルを処理しました．")
    print(f"作成されたフォルダ: {dest_dir}")

if __name__ == "__main__":
    # ▼ここに変換したいフォルダのパスを指定します▼
    # "./" は現在のディレクトリ（カレントディレクトリ）を指します
    TARGET_DIR = "./tex_sotsuron" 
    
    # 実行
    if os.path.exists(TARGET_DIR):
        process_thesis_materials(TARGET_DIR)
    else:
        print(f"エラー: 指定されたフォルダ '{TARGET_DIR}' が見つかりません．")
        print("スクリプトファイルが 'tex_sotsuron' フォルダと同じ階層にあるか確認してください．")