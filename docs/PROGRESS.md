# 統合進行計画票

> MEGA/ 配下の全プロジェクト進行状況

---

## 凡例

| 記号 | 意味 |
|---|---|
| ✅ | 完了 |
| 🔧 | 実装中 |
| ⏳ | 未着手 |
| 📝 | 設計完了 (PRD/ADR/PLAN) |

---

## UI 優先順位

```
★ 第1位: melon-chat (LINE 風チャット UI)
   独立プロジェクト。全サービスの最前面。
   画像送信→認識→会話の往復を1つのUIで。
   スマホ最適化。REST の request/response がそのまま会話に。
   Go + SQLite + SSE。

☆ 第2位: LED Board (led-board)
   補助表示。melon-chat の結果を部屋の LED にミラーするだけ。
   なくても全サービスは成立する。
```

---

## アーキテクチャ (v2: melon-chat Agent)

```
```
                  ┌───────────────────────────────────────┐
                  │       melon-chat (Agent / Router)      │ ← ★ メインUI
                  │  Go + SQLite + SSE                     │
                  │  POST /api/chat                        │
                  │  Content-Type/文脈で自動ルーティング      │
                  └──────┬───────┬───────┬────────┬────────┘
                         │       │       │        │
              ┌──────────┘       │       │        └──────────────┐
              │                  │       │                       │
              ▼                  ▼       ▼                       ▼
   ┌──────────────┐  ┌──────────────┐  ┌──────────┐  ┌──────────────────┐
   │  jacket-eye   │  │  winox       │  │ news-    │  │  外部API         │
   │  🎵 青        │  │  🍷 葡萄色   │  │ server   │  │  MusicBrainz     │
   │  Go           │  │  Java/Spring │  │ 📰 白    │  │  YouTube/Spotify │
   │  VLM認識      │  │  ワイン情報  │  │ RSS      │  │  サンプル検索     │
   │  :8085        │  │  :8087       │  │ :8081    │  │                  │
   └──────────────┘  └──────────────┘  └──────────┘  └──────────────────┘
                           │
                           ▼
                   ┌──────────────┐
                   │  Neon/Postgres│
                   │  ワインDB     │
                   └──────────────┘

   melon-sound (将来):
     music-id  :8082 🎵 fingerprint
     voice-id  :8083 🎵 話者識別
     transcribe:8084 🎵 楽譜
```
```

---

## 全サービス 状態サマリー

| サービス | ポート | 色 | 言語 | 役割 | 状態 |
|----------|--------|-----|------|------|------|
| led-board | 8080 | ⚪ | TS | 表示板 (server mode) | 🔧 |
| news-server | 8081 | 📰 | TS | RSS ニュース | ✅ |
| melon-sound/music-id | 8082 | 🎵 | Rust→WASM | fingerprint | ✅ |
| melon-sound/voice-id | 8083 | 🎵 | — | 話者識別 | 📝 |
| melon-sound/transcription | 8084 | 🎵 | — | 楽譜 | 📝 |
| jacket-eye | 8085 | 🎵 青 | Go | VLM ジャケ写認識 | ✅ |
| melon-chat | 8086 | — | Go | LINE風チャット + Agent | ✅ |
| winox | 8087 | 🍷 葡萄色 | Java/Spring | ワイン情報 | ✅ 既存 |
| dopeness | — | 🎓 | Vite/React | 掲示板/しりとり (Neon DB) | ✅ API化 ✅ |

---

## 0. 共通 API 仕様

### POST /api/message — メッセージ送信 (led-board)

```bash
curl -X POST http://localhost:8080/api/message \
  -H "Content-Type: application/json" \
  -d '{"text":"🎵 曲名 - アーティスト","source":"music-id"}'
```

| フィールド | 型 | 必須 | デフォルト | 説明 |
|---|---|---|---|---|
| `text` | string | ✅ | — | 表示するテキスト |
| `source` | string | | `"api"` | 送信元識別子（ログ用） |
| `priority` | number | | `0` | 低いほど優先（負数可） |
| `ttl` | number (ms) | | `30000` | 表示継続時間、0=永続 |
| `style` | object | | `{}` | 表示ヒント |

### GET /api/events — SSE ストリーム

```
data: {"id":"msg_...","text":"🎵 曲名","source":"music-id","priority":-1}
```

---

## 1. led-board — 表示板

| # | タスク | 状態 |
|---|---|---|
| 1.1 | メッセージキュー + priority 制御 | 🔧 |
| 1.2 | POST /api/message | 🔧 |
| 1.3 | GET /api/events (SSE) | ✅ |
| 1.4 | GET /api/health | ⏳ |
| 1.6 | DEMO モード | ✅ |
| 1.7 | CSV ログ | ✅ |
| 1.8 | カラー / Size / Glow / Matrix / Speed | ✅ |
| 1.9 | 本体モード (body.mc) | ✅ |
| 1.10-13 | static mode | ⏳ |

---

## 2. melon-sound — 音声認識

### 2.1 shared（共通ライブラリ）

| # | タスク | 状態 | 言語 |
|---|---|---|---|
| 2.1.1 | STFT / FFT | ✅ | Rust |
| 2.1.2-5 | MFCC / RingBuffer / WASAPI / リサンプル | 📝 | Rust |

### 2.2 music-id (fingerprint)

| # | タスク | 状態 |
|---|---|---|
| 2.2.1-2 | FFT / STFT / ピーク / ハッシュ / DB / WASM | ✅ |
| 2.2.3 | HTTP API 化（スタンドアロン） | ⏳ |
| 2.2.4-5 | PCM受信 / led-board 連携 | ⏳ |

### 2.3 voice-id / 2.4 transcription

| # | タスク | 状態 |
|---|---|---|
| 2.3.1-4 | VAD / MFCC+GMM / HTTP API / d-vector | 📝 |
| 2.4.1-4 | YIN / MIDI / HTTP API | 📝 |

---

## 3. jacket-eye — VLM ジャケ写認識 ✅

| # | タスク | 状態 | 言語 |
|---|---|---|---|
| 3.0 | Go サーバー実装 (Sakura AI proxy) | ✅ | Go |
| 3.1 | POST /api/jacket/scan | ✅ | Go |
| 3.2 | CLI: scan cover.jpg | ✅ | Go |
| 3.3 | Sakura AI 連携 | ✅ | Go |
| 3.4 | led-board / melon-chat 連携 | ✅ | Go (handlers.go) |

---

## 3a. melon-chat — LINE 風チャット UI + Agent ✅

| # | タスク | 優先度 | 状態 | 言語 |
|---|---|---|---|---|
| 3a.1 | LINE 風チャット UI (HTML+JS) | P0 | ✅ | Go 埋め込みテンプレート |
| 3a.2 | POST /api/chat (テキスト + 画像) | P0 | ✅ | Go |
| 3a.3 | SSE ストリーム | P1 | ✅ | Go |
| 3a.4 | SQLite 会話ログ | P1 | ✅ | Go + modernc/sqlite |
| 3a.5 | 画像アップロード | P1 | ✅ | Go |
| 3a.6 | jacket-eye 連携 (画像→認識) | P1 | ✅ | Go |
| 3a.7 | ビルド ✅ | P0 | ✅ | Go 1.25 |

### 3a.8-12: Agent 機能 (計画済み 📝)

| # | タスク | 優先度 | 状態 | 詳細 |
|---|---|---|---|---|
| 3a.8 | POST /api/agent/identify | P0 | 📝 | フルパイプライン |
| 3a.9 | inventory DB + CRUD API | P0 | 📝 | コレクション管理 |
| 3a.10 | MusicBrainz メタデータ連携 | P1 | 📝 | アルバム詳細取得 |
| 3a.11 | YouTube/Spotify サンプル検索 | P1 | 📝 | 試聴URL |
| 3a.12 | Sakura LLM 解説生成 | P1 | 📝 | アルバム解説カード |
| 3a.13 | recognition_cache | P2 | 📝 | 重複VLM呼び出し防止 |
| 3a.14 | Render デプロイ | P0 | 📝 | モバイル公開 |

---

## 4. news-server — RSS ニュース

| # | タスク | 状態 |
|---|---|---|
| 4.1 | TypeScript 試作 | ✅ |
| 4.2 | led-board から分離 | ✅ |
| 4.3 | Perl + Plagger 版 | 📝 |
| 4.4 | led-board 連携 | ✅ |

---

## 5. ネットワーク図

### ポート割り当て

```
```
localhost:8080   led-board      表示板（server mode）
localhost:8081   news-server    RSS ニュース
localhost:8082   melon-sound    music-id (fingerprint)
localhost:8083   melon-sound    voice-id (話者識別)
localhost:8084   melon-sound    transcription (楽譜)
localhost:8085   jacket-eye     VLM ジャケ写認識 (🎵 青)
localhost:8086   melon-chat     LINE 風チャット UI + Agent（最前面）
localhost:8087   winox          ワイン情報 (🍷 葡萄色 #722f37, Spring Boot)
```
```

### 起動コマンド

```bash
# 表示板
cd led-board && npm run dev                           # → :8080

# ニュース
cd news-server && npx tsx src/news-server.ts          # → :8081

# VLM ジャケ写認識
cd jacket-eye && ./jacket-eye.exe                     # → :8085

# LINE風チャット (メイン)
cd melon-chat && ./melon-chat.exe                     # → :8086

# ワイン情報 (別リポジトリ)
cd ~/JAVA/WINOX && ./mvnw spring-boot:run             # → :8087

# その他サービスは未実装
cd melon-sound/music-id && cargo run                  # → :8082 (将来)
```

---

## 6. アーキテクチャ文書一覧

| # | 文書 | 状態 | 内容 |
|---|---|---|---|
| 1 | ADR-001 | ✅ | 曲名特定 (fingerprint) — 方式A採用 |
| 2 | ADR-002 | ✅ | 機能肥大化対策 — Lite版提案 |
| 3 | ADR-003 | ✅ | 3機能分割 + 3層アーキテクチャ |
| 4 | ADR-004 | ✅ | 言語選定 (旧) — Rust第一 |
| 5 | ADR-005 | ✅ | 言語選定 (新) — 適材適所 / Polyglot |
| 6 | ADR-006 | ✅ | 入力分岐 — 明示的BTN採用 |
| 7 | PRD-audio-intelligence | ✅ | 3機能統合PRD |
| 8 | PRD-music-fingerprint | ✅ | 方式A詳細 (fingerprint) |
| 9 | PRD-news-server | ✅ | news-server 設計 |
| 10 | PRD-jacket-vlm | ✅ | jacket-eye 設計 |
| 11 | PRD-melon-chat | ✅ | melon-chat LINE風チャット |
| 12 | PLAN-agent-architecture | ✅ | melon-chat Agent計画 (今ここ) |
| 13 | DESIGN-PHILOSOPHY | ✅ | API主役、CLI先、GUIは飾り |
| 14 | NOTES-ai-integration | ✅ | Sakura AI-engine 連携設計 |
| — | ADR-007 (Spring Boot) | 📝 | Spring Boot 統合判断 (将来) |
| 15 | DESIGN-PHILOSOPHY (更新) | ✅ | サービスの色 / melon-chat ルーター / winox 連携 |
| 16 | winox | ✅ | 既存 Spring Boot + PostgreSQL (Neon) |

---

## フェーズロードマップ

```
Phase 0: 基盤
  led-board server mode ──────── 🔧

Phase 1: 完了
  music-id (Rust WASM) ───────── ✅
  news-server (TS) ───────────── ✅
  プロジェクト分割＋文書化 ────── ✅

Phase 2: 完了
  jacket-eye (Go Sakura proxy) ─ ✅
  melon-chat (Go + SQLite + SSE) ✅

Phase 3: Agent 機能 (これから)
  agent/identify パイプライン ─── 📝
  inventory DB ────────────────── 📝
  MusicBrainz / YouTube 連携 ──── 📝
  Sakura LLM 解説生成 ─────────── 📝

Phase 4: 公開・展開
  Render デプロイ ─────────────── 📝
  dopeness Vercelデプロイ ──────── ✅ (2026-05-17)
  led-board static mode ───────── ⏳
  Spring Boot 評価 ────────────── ✅ winox で実証済み
  Plagger 移行 ────────────────── 📝
  — winox (Spring Boot) 連携:    melon-chat から HTTP 呼び出し (結合ゼロ) 📝
  — 色の継承:                    サービス色 → melon-chat → LED Board 📝

## 2026-05-17 作業ログ

### dopeness Neon DB移行 ✅
- `@neondatabase/serverless` インストール
- BoardPage.tsx: localStorage → `POST/GET/DELETE /api/board`
- ShiritoriPage.tsx: ローカル状態 → `GET/POST /api/shiritori`
- Vercel Edge Functions に `api/board.ts`, `api/shiritori.ts` 追加 + deploy
- ハマりポイント:
  - 初期 `sql = neon(process.env.DATABASE_URL)` をモジュールレベルで実行すると FUNCTION_INVOCATION_FAILED → ハンドラ内で `db()` 関数経由で遅延初期化
  - `runtime: 'edge'` 必須（serverless λ は動作せず）
  - Neon DDL は unpooled 接続で実行必要（pooled 接続は PgBouncer が DDL を抑制）
  - Vercel Neon 統合は dev/prod で別 DB ブランチ → 両方にマイグレーション実行
- Migration: Node.js + `@neondatabase/serverless` tagged template で実行
- 本番URL: https://dgra.vercel.app (alias), Edge Functions on Vercel
```

## TODO 残務

### 優先度高

| # | タスク | プロジェクト | 備考 |
|---|--------|------------|------|
| 1 | **melon-chat + jacket-eye Renderデプロイ** | melon-chat | Blueprint `render.yaml` 作成済み. Render Dashboardで `SAKURA_API_KEY`, `SAKURA_SECRET`, `DATABASE_URL` をsecret設定後にBlueprint deploy |
| 2 | **saru-project 構築** | saru-project | Ruby on Rails + AIしりとりエージェント育成 + DB構築 + サンプル学習基盤 (ユーザーリクエスト) |

### 優先度中

| # | タスク | プロジェクト | 備考 |
|---|--------|------------|------|
| 3 | melon-chat Agent機能 (`POST /api/agent/identify`) | melon-chat | MusicBrainz/YouTube連携, Sakura LLM解説生成 |
| 4 | dopeness Osharecoページ実装 | dopeness | `api/oshareco.ts` は作成済み、フロントエンド未着手 |
| 5 | Neon DB バックアップ方針 | dopeness/melon-chat | NeonのPITR / branching 活用 |
| 6 | led-board static mode | led-board | 未着手 |
