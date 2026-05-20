# melon-chat Agent Architecture Plan

## 概要

melon-chat を単なるチャット UI から **マルチドメインエージェント** へ進化させる。
ユーザーは 1 つのチャット窓口から、音楽・ワイン・ニュースを横断して扱える。

各ドメインは独立したバックエンドサービスが処理し、melon-chat は Content-Type / 文脈でルーティングするだけ。

```
サービス一覧 (各ドメインの色):

  jacket-eye (音楽認識)  🎵 青  #0f6fff  :8085
  winox (ワイン情報)     🍷 葡萄色 #722f37  :8087
  news-server (RSS)     📰 白  #ffffff  :8081
  melon-sound (音声認識) 🎵 青  #0f6fff  :8082-8084
```

### 音楽認識フロー (レコード写真)

```
ユーザー [📷 レコード写真]
  │ POST /api/chat { image }
  ▼
melon-chat agent
  ├──① jacket-eye (VLM識別) ───→ Sakura AI
  ├──② MusicBrainz API (メタデータ)
  ├──③ YouTube / Spotify (サンプル)
  ├──④ SQLite inventory (所持確認)
  └──⑤ Sakura LLM (解説生成)
  │
  ▼ SSE push (色: 🎵 青)
ユーザー [🎵 結果カード]
```

### ワイン照会フロー (テキスト)

```
ユーザー [🍷 "Chianti Classico について教えて"]
  │ POST /api/chat { text }
  ▼
melon-chat agent (文脈から「ワイン」と判断)
  │
  ├──① winox:8087/api/wines/search?q=Chianti+Classico
  │     └── Spring Boot + PostgreSQL (Neon)
  ├──② 結果をリッチカードに整形
  │
  ▼ SSE push (色: 🍷 葡萄色 #722f37)
ユーザー [🍷 ワイン詳細カード]
  ┌──────────────────────┐
  │ 🍷 ワイン情報         │
  │                      │
  │  Chianti Classico    │
  │  Antinori / 2018     │
  │  トスカーナ / 赤      │
  │  ボディ: ミディアムフル│
  │  値段: 2500円         │
  │  テイスティング:      │
  │  チェリーの香り…     │
  └──────────────────────┘
```

---

## Phase 1: 基本アーキテクチャ (今ここ)

### エンドポイント追加

| Method | Path | 説明 |
|--------|------|------|
| POST | `/api/agent/identify` | 画像→フルパイプライン (同期的に全行程) |
| GET | `/api/inventory` | コレクション一覧 |
| POST | `/api/inventory` | レコード追加 |
| DELETE | `/api/inventory/{id}` | レコード削除 |
| GET | `/api/inventory/search?q=` | 所持検索 |

### データベース追加 (SQLite)

```sql
CREATE TABLE inventory (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  artist TEXT NOT NULL,
  album TEXT NOT NULL,
  format TEXT DEFAULT 'vinyl',
  label TEXT DEFAULT '',
  catalog_number TEXT DEFAULT '',
  notes TEXT DEFAULT '',
  acquired_at DATE,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(artist, album)
);

CREATE TABLE recognition_cache (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  image_hash TEXT NOT NULL,
  artist TEXT,
  album TEXT,
  songs TEXT,       -- JSON array
  year INTEGER,
  raw_response TEXT,
  explanation TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_recog_hash ON recognition_cache(image_hash);
```

### Identify フルパイプライン (擬似コード)

```
handleIdentify(image):
  1. image_hash = sha256(image)
  2. cache hit? → return cached result

  // 並列実行可能なものは goroutine で同時起動
  3. vlm_result = jacketEye.scan(image)        // Sakura VLM
  4. metadata = musicbrainz.lookup(artist, album)
  5. sample = youtube.search(artist, album)
  6. owned = inventory.check(artist, album)

  // 説明は VLM 結果を使って LLM 生成
  7. explanation = sakuraLLM.explain(vlm_result, metadata)

  8. cache.save(image_hash, vlm_result, explanation)
  9. return {vlm_result, metadata, sample, owned, explanation}
```

### 並列実行戦略

ステップ 3-6 は互いに独立 → 最大 4 goroutine 並列。

```
main goroutine:
  POST /api/agent/identify
  │
  ├── goroutine A: jacket-eye VLM
  ├── goroutine B: MusicBrainz metadata
  ├── goroutine C: YouTube sample search
  └── goroutine D: inventory check
  │
  └── wait all → assemble → response
```

---

## Phase 2: メッセージタイプ拡張

既存の `recognition_result` を強化し、リッチカード表示に対応。

```json
{
  "id": 123,
  "conversation_id": "conv_abc",
  "role": "assistant",
  "content_type": "recognition_result",
  "content": {
    "artist": "Radiohead",
    "album": "OK Computer",
    "songs": ["Paranoid Android", "Karma Police", ...],
    "year": 1997,
    "label": "Parlophone",
    "catalog_number": "CDP 7243 8 55229 2 5",
    "explanation": "1997年リリースの名盤。...",
    "sample_url": "https://youtube.com/watch?v=...",
    "owned": true,
    "owned_since": "2023-05-15",
    "confidence": 0.95
  },
  "image_url": "/static/uploads/conv_abc_123.jpg"
}
```

### UI 拡張: レコードカード表示

```
┌──────────────────────────────────┐
│ 🎵 assistant                     │
│                                  │
│  ┌──────────┐                    │
│  │ ジャケ写  │  Radiohead         │
│  │  (画像)   │  OK Computer       │
│  │          │  1997 Parlophone   │
│  └──────────┘                    │
│                                  │
│  📝 解説:                        │
│  1997年5月にリリースされた...     │
│                                  │
│  📀 収録曲:                      │
│  1. Paranoid Android             │
│  2. Karma Police                 │
│  3. Exit Music (For a Film)      │
│  ...                             │
│                                  │
│  🎧 試聴: ▶ YouTube              │
│                                  │
│  ✅ 所持しています (2023-05-15)   │
│  [所持リストから削除]             │
└──────────────────────────────────┘
```

---

## Phase 3: 外部サービス

### MusicBrainz API

```
GET https://musicbrainz.org/ws/2/release/?query=artist:"Radiohead"%20ANDrelease:"OK%20Computer"&fmt=json
→ release-groups, tracks, label, catalog number, date
```

Rate limit: 1 req/s (use local cache + goroutine ticker)

### YouTube Data API (サンプル音源)

```
GET https://www.googleapis.com/youtube/v3/search?part=snippet&q=Radiohead+OK+Computer+full+album&type=video&maxResults=1
→ videoId → https://youtube.com/watch?v=...
```

### Spotify API (代替案)

```
GET https://api.spotify.com/v1/search?q=album:OK%20Computer%20artist:Radiohead&type=album&limit=1
→ album.tracks → preview_url (30秒試聴)
```

### Sakura LLM (解説生成)

VLM と同一モデル or 別の LLM でアルバム解説を生成。

```
Prompt:
あなたは音楽評論家です。
以下のアルバムについて、200字程度で解説してください。

アーティスト: {artist}
アルバム: {album}
リリース年: {year}
ジャンル: {genre}
収録曲: {songs}

解説には以下を含めてください:
- アルバムの音楽的特徴
- 歴史的意義
- おすすめの聴きどころ
```

---

## Phase 4: Deploy (Render)

### Render デプロイ構成

```
Service Type: Web Service
Build Command: go build -o melon-chat
Start Command: ./melon-chat
Port: 10000 (Render default → env PORT)

Environment Variables:
  MELON_CHAT_PORT=10000
  MELON_CHAT_DB=/data/melon-chat.db  ← Render Disk
  JACKET_EYE_URL=https://jacket-eye.onrender.com
  LED_BOARD_URL=  (optional, not needed on Render)
```

### Render Disk 永続化

SQLite は Render Disk (`/data`) にマウントして永続化。
無料枠: 1GB まで。

```
mount: /data
path: /data/melon-chat.db
```

### 注意点

| 項目 | 対応 |
|------|------|
| Go 単一バイナリ | ✅ 依存ゼロ、そのまま動く |
| SQLite | ✅ modernc (CGO不要) → Render でビルドOK |
| 画像アップロード | Render Disk に保存 or S3 (将来) |
| SSE | Render の HTTP → そのまま動作 (WebSocket不要) |
| コールドスタート | 無料枠は 15分 idle で休眠 → DBは Disk に残る |

### デプロイ手順

```bash
# 1. GitHub に push
git add melon-chat/
git commit -m "melon-chat: agent architecture initial"
git push

# 2. Render Dashboard
#    New Web Service → select repo → melon-chat/
#    Build: go build -o melon-chat
#    Start: ./melon-chat
#    Add Disk: /data 1GB
#    Add env: MELON_CHAT_PORT=10000
#    Deploy

# 3. jacket-eye も同様にデプロイ (必要なら)
#    Render Private Service として → melon-chat から内部通信
```

---

## Phase 5: Spring Boot 統合 (ADR 混ぜ方)

### ADR 文書としての位置づけ

新しいファイル `ADR-007-spring-boot-integration.md` を作成。

### Spring Boot の役割候補

| 役割 | 説明 | 優先度 |
|------|------|--------|
| A) 管理画面・認証 | Spring Security + Admin UI | P3 |
| B) データ分析基盤 | コレクション分析、統計、レコメンド | P3 |
| C) 外部API統合バックエンド | YouTube/Spotify/MusicBrainz の安定した連携 | P2 |
| D) 置き換え対象なし | 現状のGo + Pythonで十分 | — |

### 推奨: 役割 C (API ゲートウェイ的統合)

```
              ┌───────────────────┐
              │   melon-chat      │ Go (最前面チャット)
              └────────┬──────────┘
                       │
              ┌────────▼──────────┐
              │  Spring Boot      │ Java (外部API統合)
              │  WebClient        │
              │  Retry/CircuitBrkr│
              ├───────────────────┤
              │ MusicBrainz       │
              │ YouTube Data API  │
              │ Spotify API       │
              │ Sakura AI         │
              └───────────────────┘
```

### ADR 混ぜ方指針

1. **言語はプロジェクト単位で独立** (ADR-005 遵守)
   - Spring Boot は `melon-sound-spring-boot/` や `melon-integration/` のような独立プロジェクト
   - Go サービスからは HTTP で呼ぶだけ (ゼロ結合)

2. **新しい ADR を作成**
   - `ADR-007-spring-boot-integration.md`
   - 内容: Spring Boot を外部API統合層として採用する理由・判断基準

3. **既存 ADR との関係**
   - ADR-005 (polyglot) の精神に合致: 「適材適所」
   - Spring Boot = 重厚だけど信頼性・エコシステムが強み
   - ただし「Go で足りているところに Spring Boot を無理に導入しない」

4. **導入判断チャート**

```
外部API連携が必要？
├── Yes ─── 1-2 API だけ？ → Go で十分 (現状維持)
│          3+ API / 複雑なリトライ？ → Spring Boot 検討
├── 管理画面が必要？
│   └── Yes → Spring Boot + Vaadin/Thymeleaf
├── リアルタイム性が必要？
│   └── Yes → Go (そのまま)
└── どちらでも？
    └── チームの得意言語で選ぶ
```

---

## 実装ロードマップ

| Phase | 内容 | 期間目安 | 優先度 |
|-------|------|----------|--------|
| 1 | agent/identify エンドポイント + 並列パイプライン | 1週 | P0 |
| 1b | inventory DB + CRUD API | 2日 | P0 |
| 2 | MusicBrainz 連携 (メタデータ取得) | 2日 | P1 |
| 2b | YouTube サンプル検索 | 1日 | P1 |
| 3 | Sakura LLM 解説生成 | 2日 | P1 |
| 3b | UI リッチカード表示 (recognition_result強化) | 2日 | P1 |
| 4 | Render デプロイ | 1日 | P0 |
| 5 | recognition_cache (重複VLM呼び出し防止) | 1日 | P2 |
| 6 | Spring Boot 統合 (評価→導入判断) | 1週 | P3 |

---

## ファイル構成 (追加分)

```
melon-chat/
├── agent.go              # Agent pipeline orchestration
├── inventory.go          # Inventory CRUD + DB
├── agent_handlers.go     # /api/agent/* handlers
├── inventory_handlers.go # /api/inventory/* handlers
├── musicbrainz.go        # MusicBrainz API client
├── youtube.go            # YouTube API client
├── sakura_llm.go         # Sakura LLM explain
├── static/
│   └── css/
│       └── style.css     # + recognition_result card styles
├── static/
│   └── js/
│       └── app.js        # + agent message rendering
├── static/
│   └── index.html        # + recognition_card template
├── PLAN-agent-architecture.md   # ← 本ドキュメント
└── ADR-007-spring-boot-integration.md  # ← 将来作成
```

---

## リスクと対策

| リスク | 確率 | 影響 | 対策 |
|--------|------|------|------|
| MusicBrainz rate limit | 高 | 中 | ローカルキャッシュ + 1req/s 制御 |
| YouTube API 停止 | 低 | 高 | Spotify フォールバック |
| Sakura API 障害 | 中 | 高 | recognition_cache + 再試行 |
| VLM 誤認識 (ジャケ写≠アルバム) | 中 | 中 | 複数候補提示、ユーザー選択 |
| Render 無料枠の制限 | 高 | 低 | 15分スリープ→初回遅延のみ |
