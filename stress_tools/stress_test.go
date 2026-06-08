package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// go test -v -timeout 30s -run=^TestLiveServerLoad$ .\stress_tools\stress_test.go
const baseURL = "http://localhost:8080"

type ErrorDetail struct {
	AccountID uint64
	Stage     string
	Message   string
}

func TestMember(t *testing.T) {
	transport := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	var totalUniqueUsers int32 = 0
	var accountTicker uint64 = 0

	var wg sync.WaitGroup
	concurrencySem := make(chan struct{}, 200)

	var failedMutex sync.Mutex
	var failedDetails []ErrorDetail

	startTime := time.Now()

	for i := 1; i <= 2000; i++ {
		wg.Add(1)
		concurrencySem <- struct{}{}

		currentID := atomic.AddUint64(&accountTicker, 1)

		go func(accountID uint64) {
			defer wg.Done()
			defer func() { <-concurrencySem }()

			// 動作一：登入驗證
			loginInput := map[string]string{
				"account":  fmt.Sprintf("admin%04d", accountID),
				"password": "admin123",
			}
			loginJson, _ := json.Marshal(loginInput)

			loginReq, err := http.NewRequest("POST", baseURL+"/sessions", bytes.NewBuffer(loginJson))
			if err != nil {
				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{AccountID: accountID, Stage: "Login", Message: fmt.Sprintf("建立請求失敗: %v", err)})
				failedMutex.Unlock()
				return
			}
			loginReq.Header.Set("Content-Type", "application/json")

			loginResp, err := client.Do(loginReq)
			if err != nil {
				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{AccountID: accountID, Stage: "Login", Message: fmt.Sprintf("網路發送失敗或超時: %v", err)})
				failedMutex.Unlock()
				return
			}

			if loginResp.StatusCode != http.StatusOK {
				// 讀取登入失敗時的錯誤訊息
				errBytes, _ := io.ReadAll(loginResp.Body)
				loginResp.Body.Close()

				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{
					AccountID: accountID,
					Stage:     "Login",
					Message:   fmt.Sprintf("登入狀態碼: %d, 錯誤內容: %s", loginResp.StatusCode, string(errBytes)),
				})
				failedMutex.Unlock()
				return
			}

			bodyBytes, _ := io.ReadAll(loginResp.Body)
			loginResp.Body.Close()

			var loginResult map[string]interface{}
			_ = json.Unmarshal(bodyBytes, &loginResult)

			var accessToken string
			if bodyMap, hasBody := loginResult["body"].(map[string]interface{}); hasBody {
				accessToken, _ = bodyMap["access_token"].(string)
			} else if dataMap, hasData := loginResult["data"].(map[string]interface{}); hasData {
				accessToken, _ = dataMap["access_token"].(string)
			} else {
				accessToken, _ = loginResult["access_token"].(string)
			}

			if accessToken == "" {
				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{AccountID: accountID, Stage: "Login", Message: "回應中無 Token 欄位"})
				failedMutex.Unlock()
				return
			}

			// 打MEMBER API
			memberReq, _ := http.NewRequest("GET", baseURL+"/member", nil)
			memberReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
			memberReq.Header.Set("Content-Type", "application/json")

			memberResp, err := client.Do(memberReq)
			if err != nil {
				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{AccountID: accountID, Stage: "Business", Message: fmt.Sprintf("請求商務層失敗: %v", err)})
				failedMutex.Unlock()
				return
			}

			if memberResp.StatusCode == http.StatusOK {
				currentTotal := atomic.AddInt32(&totalUniqueUsers, 1)
				fmt.Printf("[%04d/2000] 帳號 admin%04d 成功登入 [/member] (200 OK)\n",
					currentTotal, accountID)
				_, _ = io.Copy(io.Discard, memberResp.Body)
				memberResp.Body.Close()
			} else {
				// 讀取商務層 500 錯誤時，Server 回傳的具體錯誤原因
				errBytes, _ := io.ReadAll(memberResp.Body)
				memberResp.Body.Close()

				fmt.Printf("[ALERT] 帳號 admin%04d 商務層異常狀態碼: %d\n",
					accountID, memberResp.StatusCode)

				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{
					AccountID: accountID,
					Stage:     "Business",
					Message:   fmt.Sprintf("狀態碼: %d, 錯誤內容: %s", memberResp.StatusCode, string(errBytes)),
				})
				failedMutex.Unlock()
			}

		}(currentID)
	}

	wg.Wait()

	fmt.Println("\n--------------------------------------------------")
	fmt.Printf("測試結束報告:\n")
	fmt.Printf("成功通車不重複用戶: %d/2000\n", totalUniqueUsers)
	fmt.Printf("失敗用戶總計: %d/2000\n", len(failedDetails))
	fmt.Printf("總共耗時: %v\n", time.Since(startTime))
	fmt.Println("--------------------------------------------------")

	if len(failedDetails) > 0 {
		fmt.Println("以下為遭到淘汰的失敗帳號與具體錯誤明細:")
		for _, detail := range failedDetails {
			fmt.Printf("-> 帳號: admin%04d | 發生階段: %-8s | %s\n",
				detail.AccountID, detail.Stage, detail.Message)
		}
		fmt.Println("--------------------------------------------------")
	} else {
		fmt.Println("100% 成功！")
		fmt.Println("--------------------------------------------------")
	}
}
func TestMemberMQ(t *testing.T) {
	transport := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	var totalUniqueUsers int32 = 0
	var accountTicker uint64 = 0

	var wg sync.WaitGroup
	concurrencySem := make(chan struct{}, 200)

	var failedMutex sync.Mutex
	var failedDetails []ErrorDetail

	startTime := time.Now()

	for i := 1; i <= 2000; i++ {
		wg.Add(1)
		concurrencySem <- struct{}{}

		currentID := atomic.AddUint64(&accountTicker, 1)

		go func(accountID uint64) {
			defer wg.Done()
			defer func() { <-concurrencySem }()

			// 動作一：登入驗證
			loginInput := map[string]string{
				"account":  fmt.Sprintf("admin%04d", accountID),
				"password": "admin123",
			}
			loginJson, _ := json.Marshal(loginInput)

			loginReq, err := http.NewRequest("POST", baseURL+"/sessions", bytes.NewBuffer(loginJson))
			if err != nil {
				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{AccountID: accountID, Stage: "Login", Message: fmt.Sprintf("建立請求失敗: %v", err)})
				failedMutex.Unlock()
				return
			}
			loginReq.Header.Set("Content-Type", "application/json")

			loginResp, err := client.Do(loginReq)
			if err != nil {
				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{AccountID: accountID, Stage: "Login", Message: fmt.Sprintf("網路發送失敗或超時: %v", err)})
				failedMutex.Unlock()
				return
			}

			if loginResp.StatusCode != http.StatusOK {
				// 讀取登入失敗時的錯誤訊息
				errBytes, _ := io.ReadAll(loginResp.Body)
				loginResp.Body.Close()

				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{
					AccountID: accountID,
					Stage:     "Login",
					Message:   fmt.Sprintf("登入狀態碼: %d, 錯誤內容: %s", loginResp.StatusCode, string(errBytes)),
				})
				failedMutex.Unlock()
				return
			}

			bodyBytes, _ := io.ReadAll(loginResp.Body)
			loginResp.Body.Close()

			var loginResult map[string]interface{}
			_ = json.Unmarshal(bodyBytes, &loginResult)

			var accessToken string
			if bodyMap, hasBody := loginResult["body"].(map[string]interface{}); hasBody {
				accessToken, _ = bodyMap["access_token"].(string)
			} else if dataMap, hasData := loginResult["data"].(map[string]interface{}); hasData {
				accessToken, _ = dataMap["access_token"].(string)
			} else {
				accessToken, _ = loginResult["access_token"].(string)
			}

			if accessToken == "" {
				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{AccountID: accountID, Stage: "Login", Message: "回應中無 Token 欄位"})
				failedMutex.Unlock()
				return
			}

			// 打MEMBER API
			memberReq, _ := http.NewRequest("GET", baseURL+"/member-mq", nil)
			memberReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
			memberReq.Header.Set("Content-Type", "application/json")

			memberResp, err := client.Do(memberReq)
			if err != nil {
				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{AccountID: accountID, Stage: "Business", Message: fmt.Sprintf("請求商務層失敗: %v", err)})
				failedMutex.Unlock()
				return
			}

			if memberResp.StatusCode == http.StatusOK {
				currentTotal := atomic.AddInt32(&totalUniqueUsers, 1)
				fmt.Printf("[%04d/2000] 帳號 admin%04d 成功登入 [/member] (200 OK)\n",
					currentTotal, accountID)
				_, _ = io.Copy(io.Discard, memberResp.Body)
				memberResp.Body.Close()
			} else {
				// 讀取商務層 500 錯誤時，Server 回傳的具體錯誤原因
				errBytes, _ := io.ReadAll(memberResp.Body)
				memberResp.Body.Close()

				fmt.Printf("[ALERT] 帳號 admin%04d 商務層異常狀態碼: %d\n",
					accountID, memberResp.StatusCode)

				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{
					AccountID: accountID,
					Stage:     "Business",
					Message:   fmt.Sprintf("狀態碼: %d, 錯誤內容: %s", memberResp.StatusCode, string(errBytes)),
				})
				failedMutex.Unlock()
			}

		}(currentID)
	}

	wg.Wait()

	fmt.Println("\n--------------------------------------------------")
	fmt.Printf("測試結束報告:\n")
	fmt.Printf("成功通車不重複用戶: %d/2000\n", totalUniqueUsers)
	fmt.Printf("失敗用戶總計: %d/2000\n", len(failedDetails))
	fmt.Printf("總共耗時: %v\n", time.Since(startTime))
	fmt.Println("--------------------------------------------------")

	if len(failedDetails) > 0 {
		fmt.Println("以下為遭到淘汰的失敗帳號與具體錯誤明細:")
		for _, detail := range failedDetails {
			fmt.Printf("-> 帳號: admin%04d | 發生階段: %-8s | %s\n",
				detail.AccountID, detail.Stage, detail.Message)
		}
		fmt.Println("--------------------------------------------------")
	} else {
		fmt.Println("100% 成功！")
		fmt.Println("--------------------------------------------------")
	}
}


func TestTradeMQ(t *testing.T) {    
	transport := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		IdleConnTimeout:     90 * time.Second, // 延長閒置連線釋放時間
		DisableKeepAlives:   false,            // 確保開啟複用連線機制
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second, // 將超時防線由十秒放寬至十五秒防範極限排隊
	}

	var totalSuccessOrders int32 = 0
	var accountTicker uint64 = 0

	var wg sync.WaitGroup
	concurrencySem := make(chan struct{}, 200)

	var failedMutex sync.Mutex
	var failedDetails []ErrorDetail

	startTime := time.Now()

	for i := 1; i <= 2000; i++ {
		wg.Add(1)
		concurrencySem <- struct{}{}

		currentID := atomic.AddUint64(&accountTicker, 1)

		go func(accountID uint64) {
			defer wg.Done()
			defer func() { <-concurrencySem }()

			loginInput := map[string]string{
				"account":  fmt.Sprintf("admin%04d", accountID),
				"password": "admin123",
			}
			loginJson, _ := json.Marshal(loginInput)

			loginReq, err := http.NewRequest("POST", baseURL+"/sessions", bytes.NewBuffer(loginJson))
			if err != nil {
				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{AccountID: accountID, Stage: "Login", Message: fmt.Sprintf("建立請求失敗: %v", err)})
				failedMutex.Unlock()
				return
			}
			loginReq.Header.Set("Content-Type", "application/json")
			loginReq.Close = false // 強烈要求底層複用 TCP 連線防範埠口耗盡

			loginResp, err := client.Do(loginReq)
			if err != nil {
				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{AccountID: accountID, Stage: "Login", Message: fmt.Sprintf("網路發送失敗或超時: %v", err)})
				failedMutex.Unlock()
				return
			}

			if loginResp.StatusCode != http.StatusOK {
				errBytes, _ := io.ReadAll(loginResp.Body)
				loginResp.Body.Close()

				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{
					AccountID: accountID,
					Stage:     "Login",
					Message:   fmt.Sprintf("登入狀態碼: %d, 錯誤內容: %s", loginResp.StatusCode, string(errBytes)),
				})
				failedMutex.Unlock()
				return
			}

			bodyBytes, _ := io.ReadAll(loginResp.Body)
			loginResp.Body.Close() // 必須立即關閉以利連線池回收資源

			var loginResult map[string]interface{}
			_ = json.Unmarshal(bodyBytes, &loginResult)

			var accessToken string
			if bodyMap, hasBody := loginResult["body"].(map[string]interface{}); hasBody {
				accessToken, _ = bodyMap["access_token"].(string)
			} else if dataMap, hasData := loginResult["data"].(map[string]interface{}); hasData {
				accessToken, _ = dataMap["access_token"].(string)
			} else {
				accessToken, _ = loginResult["access_token"].(string)
			}

			if accessToken == "" {
				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{AccountID: accountID, Stage: "Login", Message: "回應中無 Token 欄位"})
				failedMutex.Unlock()
				return
			}

			assignedProductID := (accountID % 4) + 1

			tradeInput := []map[string]interface{}{
				{
					"product_id": assignedProductID,
					"quantity":   1,
					"type":       "pickup",
				},
			}
			tradeJson, _ := json.Marshal(tradeInput)

			tradeReq, _ := http.NewRequest("POST", baseURL+"/trade", bytes.NewBuffer(tradeJson))
			tradeReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
			tradeReq.Header.Set("Content-Type", "application/json")
			tradeReq.Close = false // 同步確保交易連線持久複用

			tradeResp, err := client.Do(tradeReq)
			if err != nil {
				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{AccountID: accountID, Stage: "Trade", Message: fmt.Sprintf("請求交易層失敗: %v", err)})
				failedMutex.Unlock()
				return
			}

			if tradeResp.StatusCode == http.StatusOK {
				currentTotal := atomic.AddInt32(&totalSuccessOrders, 1)
				fmt.Printf("[%04d/2000] 帳號 admin%04d 成功提交訂單，搶購商品 ID %d [/trade] (200 OK)\n",
					currentTotal, accountID, assignedProductID)
				_, _ = io.Copy(io.Discard, tradeResp.Body)
				tradeResp.Body.Close() // 確保關閉讀取流釋放 TCP 資源
			} else {
				errBytes, _ := io.ReadAll(tradeResp.Body)
				tradeResp.Body.Close()

				fmt.Printf("[ALERT] 帳號 admin%04d 交易層攔截狀態碼: %d，搶購商品 ID %d 失敗\n",
					accountID, tradeResp.StatusCode, assignedProductID)

				failedMutex.Lock()
				failedDetails = append(failedDetails, ErrorDetail{
					AccountID: accountID,
					Stage:     "Trade",
					Message:   fmt.Sprintf("商品 ID %d 狀態碼: %d, 錯誤內容: %s", assignedProductID, tradeResp.StatusCode, string(errBytes)),
				})
				failedMutex.Unlock()
			}

		}(currentID)
	}

	wg.Wait()

	fmt.Println("\n--------------------------------------------------")
	fmt.Printf("交易高並發測試結束報告:\n")
	// 修正語法錯誤傳入對應的整數變數
	fmt.Printf("前線成功進單不重複用戶: %d/2000\n", totalSuccessOrders)
	fmt.Printf("交易攔截或失敗用戶總計: %d/2000\n", len(failedDetails))
	fmt.Printf("總共耗時: %v\n", time.Since(startTime))
	fmt.Println("--------------------------------------------------")

	if len(failedDetails) > 0 {
		fmt.Println("以下為交易遭到攔截淘汰的失敗帳號與具體錯誤明細:")
		for _, detail := range failedDetails {
			fmt.Printf("-> 帳號: admin%04d | 發生階段: %-8s | %s\n",
				detail.AccountID, detail.Stage, detail.Message)
		}
		fmt.Println("--------------------------------------------------")
	} else {
		fmt.Println("100% 全數完美成交！")
		fmt.Println("--------------------------------------------------")
	}
}