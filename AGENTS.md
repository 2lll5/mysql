# Agent Instructions

## Google Drive Build 交付規範（所有 Agent 必須遵守）
- GitHub Repo 專屬 Google Drive 位置：`AI Project/mysql/`（資料夾名和 Repo 名完全相同；AI Project folder ID `1Yhfro6Ss5TSCXCVIZIHEu5h7V1gL8b27`）。不可將 Build 直接散放在 `AI Project/` 頂層。
- **最新完整、已經編譯與測試的 Build 直接放專案資料夾根目錄**。ZIP／安裝包、SHA-256、所有分割檔及必要說明／還原腳本放在一起；不同平台最新交付檔可並存。上傳後核對 Drive 的檔名、大小、parent、checksum；缺 ZIP／分卷的上傳不得取代前一完整版本。
- **只在有重大功能、架構、相容性或美術差異時新建 `vX.Y.Z-說明/` 版本子資料夾**；一般舊版移到 `歷史封存/`，不要每次 Build 新增資料夾，且不可刪除舊版。
- 開發變更提交／推送 GitHub；昂貴 Actions Build/Release 優先手動 `workflow_dispatch` 或正式 `release: published`，不要每次推 main 執行。回報最新版本、Drive 資料夾連結、檔案清單及測試／校驗狀態。
**English for Codex:** Verified latest builds belong directly at `AI Project/mysql/`, material milestones only under `vX.Y.Z-description/`, routine older builds in `歷史封存/`; never promote partial uploads.
