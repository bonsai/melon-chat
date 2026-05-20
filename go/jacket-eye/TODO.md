# jacket-eye

アルバムジャケット画像を VLM で認識し、アーティスト名・曲名を特定する。

## 進捗

- [x] PRD 作成
- [x] Go サーバー実装 (Sakura AI proxy)
- [x] POST /api/jacket/scan
- [x] CLI: jacket-eye scan cover.jpg
- [x] ビルド ✅ → jacket-eye.exe
- [ ] VLM モデル選定（Qwen2-VL-2B / moondream2）→ Sakura AI で代替
- [ ] 画面自動キャプチャ連携
- [ ] melon-sound fingerprint との連携
- [ ] Render デプロイ
