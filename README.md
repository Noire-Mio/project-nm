# 多租戶交易與微服務隔離架構 (Project NM)
Project-NM 是一個用 Golang 撰寫的基礎建設模擬專案，涵蓋交易流程設計、gRPC 通訊、Token 權限控管，以及多 Schema 資料切分，適合用於後端系統架構與安全設計的學習與實作。

功能特色

gRPC 通訊流程

Token 驗證與權限控管

多 Schema 切分設計

高併發交易模擬


## 核心架構與基礎建設
### 1. 資料庫多租戶物理隔離 (Multi-Schema Isolation)
為落實嚴格的租戶數據安全與合規，系統採用 PostgreSQL 結構隔離方案。

在單一物理資料庫執行體內，為各獨立租戶（例如 tenant_001、tenant_002、tenant_003）動態建立專屬的獨立 Schema，

從底層硬碟結構上封鎖跨租戶數據越權訪問。

### 2. 動態 Context 快照傳遞與工廠注入
身分憑證封裝：當使用者通過帳密驗證後，系統將其使用者 ID、帳號以及所屬租戶（Schema）打包加密成 JWT（Access Token）回傳。

無狀態 Context 傳遞：當使用者發起請求（POST /trade）時，系統中間件會攔截 Token 並解密，將取得的 Schema 和使用者 ID 灌進 Go 的自訂上下文結構體 contexts.Trade 之中。

避免資料交叉感染：交易服務層不使用全域變數，一律由 Context 讀取當前操作的 Schema，並透過工廠模式動態拼接 SQL 語句（例如 schema.table_name）。確保在高併發、多執行緒的環境下，資料具備完全的隔離性。

### 3. 基礎建設環境部署
專案全面容器化，將 Nginx 網關、Go 主機、PostgreSQL 和 Redis 全部的通訊線路收容進 Docker 的虛擬網路（nm-network）中，從網路層面完成硬體隔離。

### 。一鍵建立並在背景運行所有服務
```
docker compose up -d --build
```
### 。檢查目前所有容器的運作狀態與連接埠對齊狀況
```
docker compose ps
```
### 。即時監看 Go 後端與背景 Worker 輸出的結構化日誌
```
docker compose logs -f app
```
### 。關閉系統並安全釋放網路與記憶體資源
```
docker compose down
```

### 4. 單連接埠多協議多路複用 (cmux Pipeline)
在微服務架構中，若同時提供 Web API 給前端，又要在系統內部跑 gRPC 通訊，通常需要佔用兩個不同的實體連接埠（例如 8080 和 50051）。

本專案引進 cmux 技術，使整個應用只需監聽一個 8080 連接埠。當流量進來時，系統會自動檢查網路請求的特徵碼：

若標頭顯示 content-type 為 application/grpc（HTTP/2），則導向 gRPC 伺服器。

若為傳統的 HTTP/1.1 請求，則導向 Gin 框架的 HTTP 伺服器。

這讓前端的 Nginx 網關配置變得非常乾淨，只需對齊一個 8080 連接埠就能同時相容兩種協議。

### 5. 集中式生命週期管理與優雅停機 (Graceful Shutdown)
集中管理 (WorkerManager)：系統在啟動時會透過 workerManager 把負責初始化會員的 MemberInitWorker 以及負責交易落盤的 TradeWorker 統一註冊控管。這些背景 Worker 會共享同一個上下文（Context）訊號，確保後台行為完全可控。

拉鏈式停機機制：利用了 signal.NotifyContext 監聽作業系統的 SIGINT 和 SIGTERM 訊號。當收到關機指令時，系統會按部就班地安全停機：

通知工人停手 (StopAll)：背景 Worker 會立刻收到取消訊號，停止從 Redis Stream 搬運新任務。但會堅持將手頭上那最後一筆正在執行的任務完全跑完，並執行 XDel 劃掉訊息，隨後才退出。

網路大門立即關閉 (GracefulStop)：gRPC 伺服器立刻停止接收新的微服務請求，並等待在線上的連線處理完畢。

5 秒超時安全防線 (Shutdown)：系統留給 Gin HTTP 伺服器 5 秒鐘的最後清空時間。如果 Worker 因為某些極端原因（例如資料庫卡死）在 5 秒內做不完，資料庫的事務會自動觸發回滾（Rollback），且因為尚未執行 XDel，該筆任務會安全地保留在 Redis Stream 隊列中，等下次重啟後自動提貨恢復。

## 核心商務模組與物理防線設計

### 1. 多租戶 Schema 自動化遷移機制
自動化遷移引擎 RunMigration 在系統啟動時（即 Serve 函數呼叫 a.Migrate 的時間點）會自動執行，確保資料庫地基完整建立：

公共區與租戶區的「物理雙層切換」：

公共資料庫維護 (public)：系統先將搜尋路徑切換到 SET search_path TO "public"，在這裡建立全系統共用的 User 表（用來管理系統管理員或跨租戶的基本帳號驗證）。

動態多租戶巡迴建立 (targetSchemas)：接著系統遍歷所有預設的租戶（如 tenant_001, tenant_002, tenant_003），若發現該租戶不存在，會直接物理建立 CREATE SCHEMA IF NOT EXISTS，並將搜尋路徑切換進去。

基於 search_path 的動態表格植入：當系統切換搜尋路徑（SET search_path TO "%s"）進入某個租戶空間後，系統會呼叫同樣的腳本，在各Schema下分別建立會員表、交易流水帳表與商品表。

防重複執行鎖 (Idempotent Record)：為避免系統每次重啟時腳本重複執行，導致原有高頻交易資料被洗掉或噴出 Table already exists 錯誤，專案在每個租戶的 Schema 內部，皆先執行 AutoMigrate(&MigrateRecord{}) 建立一張「遷移歷史紀錄表」。每次執行建表前，會先到當前租戶的紀錄表撈出歷史序號，若發現如 20260513-001（建立會員表和交易表）已經跑過，就會直接跳過；若沒跑過，跑完後立刻寫入歷史，確保每個租戶的升級進度各自獨立。

### 2. 安全認證門戶與時間戳記防線 (POST /sessions)
憑證分流機制：當使用者登入被觸發時，系統會先去資料庫驗證密碼。組裝 UserInfo 時，將 UserID、Schema（租戶名稱）以及拿到的 Permissions（權限）一起打包塞進 JWT 裡，確保使用者接下來的每一次高頻請求皆貼著這個 Token 來通行。

Redis 快取時間防雪崩：在將 Token 寫入 Redis 短效快取時，過期時間沒有設成固定的 30 分鐘，而是加上了一個隨機變數 30 + rand.Intn(5)（30 到 35 分鐘隨機跳動），從而有效打散失效時間點，防止快取雪崩。

長短效雙 Token 綁定與最新時間戳記鎖：

長短效分離：系統簽發一個 30 分鐘的短效 AccessToken 丟給前端，並在 Redis 裡綁定一個隨機產生、效期高達 7 天的長效 RefreshToken（UUID）。

登入時間戳記鎖 (LoginAt)：將使用者登入的微秒級時間戳記 userInfo.LoginAt 綁進 Redis 的 BindUserLatestLoginTime。如果同一個帳號在別的地方重複登入，生成了全新的時間戳記，舊 Token 在中途檢查時，只要發現時間戳記跟 Redis 裡最新的對不上，就會當場失效。同時，長效的 RefreshToken 讓使用者在短效 Token 到期時可以無感刷新。

### 3. 權限刷新 (PUT /sessions/refresh)
當使用者的短效 AccessToken 過期時，前端會帶當初拿到的長效 UUID（RefreshToken）來呼叫此 API：

系統會去 Redis 撈出該用戶目前在全網唯一的最新登入時間與合法鑰匙。

如果 Redis 裡的紀錄不存在（例如過期或登出），或者前端傳來的 UUID 密鑰跟 Redis 目前鎖定的對不上，當場直接攔截並回傳憑證已失效。

這能確保即便外部流出過期的舊 RefreshToken，也絕對無法穿透這道防線。最終系統會撈取用戶當下的最新狀態與最新權限，組成全新的 AccessToken 回傳給前端。

### 4. 錢包 (GET /member-mq)
當請求 GET /member-mq 時，系統先在 Transport 層執行身分驗證和權限驗證，通關後才會繼續往下執行：

快取命中流：第一步先拿 Token 裡解出來的 MemberSchema 與 UserID 去 Redis 快取撈資料，如果快取命中，就直接把會員結構體回傳。

動態回補與異步初始化：如果快取找不到，系統會動態拼出該租戶的資料表去撈取資料，並回補到 Redis 快取 30 分鐘。當使用者第一次進來查詢資產時，如果他還沒有會員資料，系統會立刻幫他虛擬建立一個錢包（給予初始餘額），並把初始化任務推入隊列，交由背景去落盤，不讓前端使用者卡住。

### 5. 交易 (POST /trade)
為重現微服務分散式溝通，本專案在同一個主機上設計了兩套方法：ProcessOrder（扮演轉發的網關客戶端）與 ExecuteOrder（扮演真正的微服務核心）。

網關計價與隔離傳遞 (Price Snapshot Isolation)
當使用者發送 POST /trade 時，系統會先進入 ProcessOrder。ProcessOrder 把前端傳來的購物車清單（dtos）與解密出來的租戶使用者資訊（ctx.UserInfo），封裝成 Protobuf 格式，透過初始化的 NewProjectNMGrpcClient，將這個請求跨網路發射出去。此時下單金額被拍下歷史快照定格，後續無論商家如何修改商品主表定價，交易金額絕不影響。

分散式鎖與記憶體層面攔截
ExecuteOrder 當流量繞了 gRPC 一圈進入核心後，利用 Redis 的 SETNX 機制，針對特定租戶的特定用戶鎖定 5 秒鐘，並用 defer 確保不論成功失敗都會釋放鎖。當同一個使用者因為網路延遲而連續點擊結帳時，這個鎖會在記憶體層面直接把後續的重複請求「當場拋棄」，從源頭杜絕重複扣款。

## TestTradeMQ
```
go test -v -timeout 30s -run=^TestTradeMQ$ .\stress_tools\stress_test.go
```
