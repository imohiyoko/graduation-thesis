graph TD
    Start(["反復ステップ開始"]) --> CheckDistinct{"x, w, v は<br/>互いに異なる点か？<br/>(放物線が作れるか？)"}

    %% 分岐判定
    CheckDistinct -- いいえ --> GoldenSection
    CheckDistinct -- はい --> TryParabola["放物線補間の試行<br/>(x, w, v を使用して p, q を計算)"]

    TryParabola --> JudgeParabola{"審判 e による判定<br/>1. 予測点は区間 [a,b] の中か？<br/>2. 動きは前回の動き e の半分以下か？"}

    %% 判定結果によるルート分岐
    JudgeParabola -- "合格 (高速ルート)" --> DoParabola["放物線補間を採用<br/>d = p / q"]
    JudgeParabola -- "不合格 (却下)" --> GoldenSection["黄金分割探索を採用<br/>(a, b, x を使用)"]

    %% 黄金分割ルート
    GoldenSection --> DoGolden["d = 黄金比によるステップ"]

    %% 合流と更新
    DoParabola --> UpdateE["変数 e の更新<br/>(今回の d を次回の判定用 e として保存)"]
    DoGolden --> UpdateE

    UpdateE --> CalcU["新しい点 u の決定<br/>u = x + d"]

    CalcU --> End(["関数の評価へ"])

    %% スタイルの定義
    style Start fill:#f9f,stroke:#333,stroke-width:2px
    style End fill:#f9f,stroke:#333,stroke-width:2px
    style JudgeParabola fill:#ff9,stroke:#f66,stroke-width:4px
    style DoParabola fill:#9f9,stroke:#333
    style DoGolden fill:#9cf,stroke:#333