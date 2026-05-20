# PRD: Jacket Eye — アルバムジャケット VLM 認識

## 位置づけ

**バックエンド API サービスの 1 つ。独自 UI は持たない。**
ユーザーは `melon-chat` (LINE 風チャット) または CLI から利用する。

## 概要

メディアプレイヤーや YouTube に表示されているアルバムジャケット画像を
Sakura AI-engine (VLM) に渡し、アーティスト名・曲名を特定する。

```
画像（スクショ / drag & drop / 画面キャプチャ）
    ↓
VLM（LLaVA / Qwen-VL / Florence-2 / moondream2）
    ↓
"アーティスト: 〇〇 / 曲名: 〇〇"
    ↓
LED Board に表示
```

## ユースケース

| シチュエーション | 方法 |
|---|---|
| PC で音楽再生中 | 画面のアルバムアート部分を自動キャプチャ |
| スマホで撮影 | 写真を drag & drop |
| ストリーミング画面 | スクリーンショットを送信 |
| fingerprint 非対応曲 | 方式A の補助として手動で画像送信 |

## VLM モデル選定

### 要件
- オフライン実行（外部API不使用）
- CPU のみで動作（CUDA 非必須）
- アルバムジャケットからアーティスト名・アルバム名を抽出可能

### 候補

| モデル | サイズ | CPU推論 | 日本語 | 精度 |
|---|---|---|---|---|
| **moondream2** | 1.6B | ✅ 軽量 | △ | ○ |
| **Qwen2-VL-2B** | 2B | ✅ | ✅ | ◎ |
| **Florence-2** | 0.23B | ✅ 最軽量 | × | ○ |
| **LLaVA-1.6** | 7B+ | △ 重い | △ | ◎ |
| **PaliGemma** | 3B | △ | × | ◎ |

**第1候補**: Qwen2-VL-2B（日本語対応 + 軽量 + 高精度）  
**第2候補**: moondream2（軽量、英語のみだが十分）  
**軽量特化**: Florence-2（232M、CPU でも高速）

### 推論エンジン

| エンジン | 対応モデル | 備考 |
|---|---|---|
| llama.cpp | LLaVA, Qwen-VL | CPU 最適、量子化対応 |
 | Ollama | LLaVA, moondream2 | 簡単セットアップ |
| mlx | 一部 | macOS 専用 |

## 処理パイプライン

```
Phase 1: 手動入力（MVP）
  画像ファイル drag & drop / ファイル選択
    → VLM 推論
    → 結果パース（JSON / テキスト）
    → LED Board に POST

Phase 2: 画面自動キャプチャ
  スクリーンショット定時取得
    → アルバムアート領域検出（object detection or 固定領域）
    → VLM 推論
    → 自動表示

Phase 3: 方式A（fingerprint）との連携
  fingerprint で未認識
    → 画面キャプチャ実行
    → VLM 推論
    → 結果を fingerprint DB に登録（次回から fingerprint で認識）
```

## プロンプト設計

```text
You are a music recognition assistant.
Analyze this album cover image and respond in JSON:
{
  "artist": "artist name",
  "album": "album name",
  "songs": ["song1", "song2", ...],
  "year": 2024
}
If uncertain, set fields to null.
```

日本語版:
```text
このアルバムジャケット画像を解析し、以下のJSONで回答してください：
{
  "artist": "アーティスト名",
  "album": "アルバム名",
  "songs": ["曲名1", "曲名2", ...],
  "year": 2024
}
不明な項目は null にしてください。
```

## 連携インターフェース

### CLI
```bash
# 画像ファイルから認識
jacket-eye scan cover.jpg

# 結果を LED Board に送信
jacket-eye scan cover.jpg --post http://localhost:8080/api/message
```

### HTTP API
```
POST /api/jacket-eye/scan
  Content-Type: multipart/form-data
  file: <image>

Response:
  { "artist": "...", "album": "...", "songs": [...], "confidence": 0.95 }
```

### LED Board 連携
```bash
curl -X POST http://localhost:8080/api/message \
  -d "text=🎵 曲名 - アーティスト"
```

## 実装計画

| Phase | 内容 | 成果物 |
|---|---|---|
| 1 | moondream2 / Qwen-VL で CLI ツール試作 | Python スクリプト |
| 2 | HTTP API 化（FastAPI） | REST サーバー |
| 3 | 画面自動キャプチャ連携 | 常駐デーモン |
| 4 | fingerprint 連携（melon-sound） | 方式A+B 統合 |

## 技術スタック（仮）

| 層 | 技術 |
|---|---|
| VLM | Qwen2-VL-2B / moondream2 (llama.cpp / Ollama) |
| API | Python FastAPI or Rust axum |
| CLI | Rust clap or Python argparse |
| 画像取得 | screenshot crate (Rust) / mss (Python) |
| 配布 | ONNX 量子化モデル + 推論エンジン同梱 |
