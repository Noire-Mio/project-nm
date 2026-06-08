POST /sessions
功能描述：會員登入與安全權限核發（壓測與交易的前置大門）。

技術細節：

接收帳號與密碼，進行資料庫比對。

校驗通過後，將用戶的核心物理資訊：UserID、Username 以及該用戶所屬的 多租戶資料庫名稱 (Schema) 共同加密封裝進 JWT 中，作為 access_token 回傳給客戶端。


DELETE /sessions
功能描述：用戶登出。

技術細節：清除客戶端的會話狀態（Session），或將該 Token 納入 Redis 黑名單，實現即時註銷。


PUT /sessions/refresh
功能描述：無感刷新 Token 權限。

技術細節：當短效期的 access_token 過期時，客戶端可攜帶長效期的 refresh_token 來換取全新 Token，避免使用者操作中斷。
