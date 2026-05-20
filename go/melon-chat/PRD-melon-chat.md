# PRD: melon-chat — LINE 風チャット UI（最優先表示）

## 位置づけ

**全サービスの最前面に立つ統一 UI。** LED Board は補助。

```
ユーザー
  │
  ▼ ★ メイン
melon-chat (LINE 風チャット)
  ├──→ jacket-eye (画像認識)
  ├──→ melon-sound (音声認識)
  ├──→ news-server (ニュース)
  ├──→ led-board (ミラー表示, オプション)
  └──→ 任意の API
```

## なぜ独立プロジェクトか

- jacket-eye 専用 UI にしない → どのサービスも同じ chat UI から使える
- どのサービスが裏で動いているかユーザーは意識しない
- 「画像送る」「音声送る」「テキストで聞く」すべて同じインターフェース
- 将来的に AI エージェントが自ら会話に参加できる

## API

### melon-chat が提供する API

| Method | Path | 説明 |
|---|---|---|
| POST | `/api/chat` | メッセージ送信 (テキスト or 画像) |
| GET | `/api/events` | SSE ストリーム（新着メッセージ） |
| GET | `/api/conversations` | 会話一覧 |
| GET | `/api/conversations/:id` | 会話詳細（メッセージ履歴） |
| DELETE | `/api/conversations/:id` | 会話削除 |

### 外部サービスが melon-chat に結果を POST

```bash
# jacket-eye が認識結果を chat に送る
curl -X POST http://localhost:8086/api/message \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "conv_123",
    "role": "assistant",
    "content_type": "recognition_result",
    "content": "🎵 曲名 - アーティスト"
  }'
```

## メッセージタイプ

| type | 表示 | 説明 |
|---|---|---|
| `text` | テキスト吹き出し | 通常の会話 |
| `image` | 画像サムネイル | ユーザーが撮影したジャケ写 etc |
| `recognition_result` | カード表示 | 曲名・アーティスト・アルバム |
| `audio` | 波形アイコン | 音声認識の入力 (将来) |
| `system` | システムメッセージ | エラー・通知 |

## DB (SQLite)

```sql
CREATE TABLE conversations (
  id TEXT PRIMARY KEY,
  title TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  conversation_id TEXT NOT NULL,
  role TEXT NOT NULL,          -- 'user' | 'assistant' | 'system'
  content_type TEXT NOT NULL,  -- 'text' | 'image' | 'recognition_result' | 'audio'
  content TEXT,                -- JSON or text
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);
```

## UI

LINE / iMessage 風の吹き出しチャット。

```
┌─────────────────────────┐
│ ← melon-chat     ＿ □ ×│
├─────────────────────────┤
│                         │
│  ┌──────────────────┐   │
│  │ 📸 ユーザー       │   │
│  │ [ジャケ写画像]    │   │
│  └──────────────────┘   │
│                         │
│  ┌──────────────────┐   │
│  │ 🎵 assistant     │   │
│  │                   │   │
│  │  アーティスト: ○○│   │
│  │  曲名: △△       │   │
│  │  アルバム: ✕✕    │   │
│  │                   │   │
│  │  推薦: 関連曲A   │   │
│  └──────────────────┘   │
│                         │
│  ┌──────────────────┐   │
│  │ ユーザー         │   │
│  │ この曲のギターTAB │   │
│  └──────────────────┘   │
│                         │
├─────────────────────────┤
│ 📷  🎤    [メッセージ]  │
└─────────────────────────┘
```

## 技術スタック

| 層 | 候補 | 理由 |
|---|---|---|
| サーバー | Go (net/http) | 軽量、単一バイナリ、テンプレート埋め込み |
| DB | SQLite (modernc) | ファイル1つ、セットアップ不要 → 後で Supabase |
| フロント | HTML + htmx or vanilla JS | SPA 不要、テンプレートで十分 |
| SSE | Server-Sent Events | 会話の自動更新 |
| 画像保存 | ローカルファイル or S3互換 | まずはローカル |

## 連携フロー

```
① ユーザーが画像をアップロード
    POST /api/chat { content_type: "image", image: ... }
② melon-chat が jacket-eye に転送
    POST /api/jacket-eye/scan { image: ... }
③ jacket-eye が Sakura AI に問い合わせ
④ 結果が melon-chat に返る
    → 会話履歴に追加
    → SSE で UI にプッシュ
    → (オプション) led-board にも POST /api/message
⑤ ユーザーがさらに質問
    POST /api/chat { text: "このバンドの他の曲は？" }
    → jacket-eye (or Sakura LLM) に転送
    → ...
```
