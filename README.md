多租戶交易系統 (Project NM)

這個專案是我為了模擬高頻率交易環境而特別開發的核心模組。
在高頻率交易的極限場景中，系統最怕遇到的就是因為幾毫秒的時間差，導致「商品賣超」或「錢包餘額扣錯帳」的致命問題。

在設計上，我捨棄了傳統比較不穩定的快取預扣作法，
而是實打實地採用了 悲觀鎖排隊 + 消息佇列（Redis Stream）幫資料庫擋子彈 + 後台 Worker 定速落盤 的防護架構，來確保每筆高頻交易的絕對安全。

資料庫物理隔離
在 PostgreSQL 裡為每個租戶（像是 tenant_001、tenant_002、tenant_003）建立獨立的 Schema。
這樣做可以從根本上確保 A 租戶的用戶絕對不會看到 B 租戶的資料，在物理層面完成徹底隔離。


Token 與 Context 的動態傳遞
登入核發 Token ，當使用者通過帳密驗證後，系統會把他的 使用者 ID、帳號 以及他所屬的 租戶名稱 (Schema) 一起打包加密成 JWT（Access Token）回傳。

當使用者發起請求（POST /trade）時，系統的中間件會攔截 Token並解密，把拿到的 Schema 和使用者 ID 灌進 Go 的自訂上下文結構體 contexts.Trade 裡面。

整個交易服務層都不使用全域變數，一律從這個 Context 讀取當前要操作的 Schema，並透過工廠模式去動態拼接 SQL 語句（例如 schema.table_name）。
這能確保在高併發、多執行緒的高頻交易環境下，確保資料不會感染。



基礎建設環境部署
我把 Nginx 網關、Go 主機、PostgreSQL 和 Redis 全部的通訊線路都收進了 Docker 的虛擬網路（nm-network）裡。


# 1. 一鍵建立並在背景跑起來所有服務
# --build 會確保你每次改完 Go 程式碼，都會重新編譯進鏡像裡，不會讀到舊檔案
docker compose up -d --build

# 2. 檢查目前所有容器的運作狀態和連接埠有沒有對齊
docker compose ps

# 3. 即時監看 Go 後端和背景 Worker 輸出的日誌
docker compose logs -f app

# 4. 關閉系統並安全释放網路與記憶體資源 
docker compose down


在微服務架構中，如果同時提供 Web API 給前端，又要在系統內部跑 gRPC 通訊，通常要佔用兩個不同的實體連接埠（例如 8080 和 50051）。

我引進了 cmux 技術。整個應用只監聽一個 8080 連接埠，當流量進來時，系統會去檢查網路請求的特徵碼。
如果標頭顯示 content-type 是 application/grpc（HTTP/2），就導向 gRPC 伺服器；
如果是傳統的 HTTP/1.1，就交給 Gin 框架的 HTTP 伺服器。
這讓前端的 Nginx 網關配置變得非常乾淨，只要對齊一個 8080 連接埠就能同時走兩種協議

集中管理 (WorkerManager)
系統在啟動時會透過 workerManager 把負責初始化會員的 MemberInitWorker 以及負責交易落盤的 TradeWorker 統一註冊控管。
這些背景 Worker 會共享同一個上下文（Context）訊號，確保後台行為完全可控。



利用了 signal.NotifyContext 監聽作業系統的 SIGINT 和 SIGTERM 訊號。
當收到關機指令時，系統會像拉鏈一樣，按部就班地安全停機：

通知工人停手 (StopAll)：背景 Worker 會立刻收到取消訊號，停止從 Redis Stream 搬運新任務，
但會堅持把手頭上那最後一筆正在執行的完全跑完，並執行 XDel 劃掉訊息，隨後才退出。

網路大門立即關閉 (GracefulStop)：gRPC 伺服器立刻停止接收新的微服務請求，並等待在線上的連線處理完畢。

5 秒超時安全防線 (Shutdown)：系統留給 Gin HTTP 伺服器 5 秒鐘的最後清空時間。
如果 Worker 因為某些極端原因（例如資料庫卡死）在 5 秒內做不完，資料庫的事務會自動觸發回滾（Rollback），
且因為還沒執行 XDel，這筆任務會安全地保留在 Redis Stream 隊列中，等下次重啟後自動提貨恢復。



多租戶動態 Schema 自動化遷移機制

RunMigration，是我設計用來讓系統啟動時自動把地基打好的自動化遷移引擎

公共區與租戶區的「物理雙層切換」
系統在啟動時（也就是前面 Serve 函數呼叫 a.Migrate 的時間點），會先執行兩件事：

公共資料庫維護 (public)：系統先將搜尋路徑切換到 SET search_path TO "public"，在這裡建立全系統共用的 User 表（用來管理系統管理員或跨租戶的基本帳號驗證）。

動態多租戶巡迴建立 (targetSchemas)：接著，系統會像巡邏車一樣，遍歷所有預設的租戶（如 tenant_001, tenant_002, tenant_003），
如果發現該租戶不存在，會直接物理建立 CREATE SCHEMA IF NOT EXISTS，並將搜尋路徑切換進去。

基於 search_path 的動態表格植入
當系統切換搜尋路徑（SET search_path TO "%s"）進入某個租戶的專屬空間後，系統會呼叫同樣的腳本：

在 tenant_001 裡執行建立會員表、交易流水帳表（CreateMemberTransactionTable）與商品表（CreateProductTable）。
切換到 tenant_002 時，再重複執行一次相同的動作。


租戶獨立的「防重複執行鎖 (execute)」
通常自動化遷移最怕系統每次重啟時，腳本又去執行一遍，導致原本已經有高頻交易資料的表被洗掉或噴出 Table already exists 錯誤。

我在每個租戶的 Schema 內部，我先 AutoMigrate(&MigrateRecord{}) 建立一張「遷移歷史紀錄表」。

每次要執行建表時，會先到當前租戶的紀錄表撈出歷史序號（loadRecords）。
如果發現像 20260513-001（建立會員表和交易表）已經跑過了，就會直接跳過。
如果沒跑過，跑完後立刻 writeRecord 寫入歷史，確保每個租戶的升級進度各自獨立。




安全認證門戶 (POST /sessions)

使用者在前端輸入帳號密碼點擊登入。因為有很多獨立的租戶，系統必須在登入這一刻就分流他是屬於哪一個 Schema 的。

當 AuthTrans.Login() 被觸發時，系統會先去資料庫驗證密碼。
組裝 UserInfo：你把 UserID、Schema (租戶名稱，如 tenant_001)、以及剛剛拿到的 Permissions (權限) 全部塞進了 JWT 裡。
接下來使用者的每一次高頻撞擊，都會貼著這個 Token 來通行

Redis 快取時間隨機跳動

在把 Token 寫入 Redis 短效快取時，過期時間沒有設成死板的 30 分鐘，而是加上了一個隨機變數 30 + rand.Intn(5)（30 到 35 分鐘隨機跳動）。

長短效雙 Token 綁定與最新時間戳記
長短效分離：系統簽發一個 30 分鐘的短效 AccessToken 丟給前端，並在 Redis 裡綁定一個隨機產生、效期高達 7 天的長效 RefreshToken（UUID）。

登入時間戳記鎖 (LoginAt)：你把使用者登入的微秒級時間戳記 userInfo.LoginAt 綁進了 Redis 的 BindUserLatestLoginTime。

如果同一個帳號在別的地方重複登入，生成了全新的時間戳記，舊 Token 在中途檢查時，只要發現時間戳記跟 Redis 裡最新的對不上，就會當場失效。
同時，長效的 RefreshToken 讓使用者在短效 Token 到期時，可以無感刷新，不需要一直重新輸入密碼。




權限刷新防線 (PUT /sessions/refresh)

當使用者的短效 AccessToken 過期時，前端會帶當初拿到的長效 UUID（RefreshToken）來呼叫這個 API。

去 Redis 撈出該用戶目前在全網唯一的最新登入時間與合法鑰匙
如果 Redis 裡的紀錄不見了( 例如過期或登出 )，或者前端傳來的 UUID 鑰匙跟 Redis 目前鎖定的對不上，當場直接噴 憑證已失效，請重新登入。
確保即便拿到了某個過期的舊 RefreshToken，也絕對無法穿透這道防線。

撈取了用戶當下的最新狀態與最新的權限組成最新的 AccessToken 回傳給前端。


取得會員 (GET /member-mq)

當請求 GET /member-mq 時，系統先在 Transport 層執行了身分驗證和權限驗證，如果通關才會繼續往下執行。

當使用者第一次進來查詢資產時，如果他還沒有會員資料，系統會立刻幫他虛擬建立一個錢包，並把丟給背景去落盤，不讓前端使用者卡住。

第一步先拿 Token 裡解出來的 MemberSchema 與 UserID 去 Redis 快取撈資料，如果快取命中，就直接把會員結構體回傳。
如果快取找不到，動態拼出該租戶的資料表去撈取資料，並回補到 Redis 快取 30 分鐘，然後把資料安全交回去。下一次他再查，就直接走快取。

T("trade", t.TradeTrans.ProcessOrder())



交易 (POST /trade)

為了重現微服務分散式溝通，
我故意在同一個主機上設計了兩套方法：ProcessOrder（扮演轉發的網關客戶端）與 ExecuteOrder（扮演真正的微服務核心）

當使用者發送 POST /trade 時，系統會先進入 ProcessOrder

ProcessOrder 把前端傳來的購物車清單（dtos）與解密出來的租戶使用者資訊（ctx.UserInfo），老老實實地封裝成 Protobuf 格式
透過 Serve 函數初始化的 NewProjectNMGrpcClient，將這個請求跨網路發射出去

ExecuteOrder 當流量繞了 gRPC 一圈，利用 Redis 的 SETNX 機制，針對特定租戶的特定用戶鎖定 5 秒鐘，並用 defer 確保不論成功失敗都會釋放鎖。
當同一個使用者因為手機卡住而瘋狂按結帳時，這個鎖會在記憶體層面直接把後續的髒請求「當場拋棄」，從源頭杜絕重複扣款。

PostgreSQL 行級悲觀鎖 (FOR UPDATE) 排隊防賣超。

當審查通過，準備要寫入時。為了不讓慢速的硬碟 I/O 把前線的 gRPC 執行緒卡死，
我使用了 Redis 的 Pipeline（管道）機制。系統不一筆一筆發送網路請求給 Redis，而是把購物車裡所有商品打包成一個大禮包，丟進 Redis Stream（stream:trade_tasks）中。
塞完立刻對外響應 pending 狀態並回傳。