# 1. 指定基礎映像檔：使用輕量級的 Alpine Linux 搭配 Go 1.21
FROM golang:1.21-alpine AS builder

# 2. 設定容器內的工作目錄
WORKDIR /app

# 3. 複製 go.mod 與 go.sum 並下載依賴，這樣可以利用 Docker 的緩存機制加速後續編譯
COPY go.mod go.sum ./
RUN go mod download

# 4. 複製專案其餘所有的程式碼
COPY . .

# 5. 編譯 Go 程式。
# CGO_ENABLED=0 確保靜態連結，減少對底層系統函式庫的依賴，適合在 Alpine 執行
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# 6. 第二階段：使用最迷你的執行環境，大幅縮小映像檔大小
FROM alpine:latest
WORKDIR /root/

# 從 builder 階段把編譯好的執行檔拿過來
COPY --from=builder /app/main .

# 暴露 8080 埠號給外部
EXPOSE 8080

# 啟動命令
CMD ["./main"]