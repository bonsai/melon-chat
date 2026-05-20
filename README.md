# melon-chat

マルチ言語チャットアプリケーションプラットフォーム

## フォルダ構成

```
melon-chat/
├── go/           # Go サービス
│   ├── jacket-eye/   # VLM ジャケ写認識 (Sakura AI proxy)
│   └── melon-chat/   # LINE風チャット UI + Agent
├── rb/           # Ruby サービス
│   └── .gitkeep
└── py/           # Python サービス
    └── .gitkeep
```

## サービス一覧

| サービス | 言語 | ポート | 説明 |
|----------|------|--------|------|
| jacket-eye | Go | 8085 | VLM ジャケ写認識 |
| melon-chat | Go | 8086 | LINE風チャット UI + Agent |

## 開発

### Go サービス

```bash
cd go/jacket-eye && go run .
cd go/melon-chat && go run .
```

### Render デプロイ

```bash
# Blueprint deploy
# Render Dashboard → Blueprint → Connect repo
# Set SAKURA_API_KEY, SAKURA_SECRET, DATABASE_URL as secret env vars
```

## 詳細

- [PROGRESS.md](docs/PROGRESS.md) - 進行状況
- [render.yaml](render.yaml) - Render デプロイ設定
