package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"project-nm/pkg/contexts"
	"project-nm/pkg/database"
	"project-nm/pkg/entities"

	"github.com/redis/go-redis/v9"
)

// SetCache 設置基礎字串快取
func SetCache(key string, value string, expiration time.Duration) error {
	return database.RDB.Set(context.Background(), key, value, expiration).Err()
}

// GetCache 獲取基礎字串快取
func GetCache(key string) (string, error) {
	return database.RDB.Get(context.Background(), key).Result()
}

// PushToStream 將資料推入 Redis Stream
func PushToStream(streamName string, data map[string]interface{}) error {
	return database.RDB.XAdd(context.Background(), &redis.XAddArgs{
		Stream: streamName,
		Values: data,
	}).Err()
}

// SetUserToken 將登入成功的 UserInfo 快取至 Redis
func SetUserToken(token string, userInfo *contexts.UserInfo, expiration time.Duration) error {
	ctx := context.Background()
	data, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}
	key := "token:" + token
	return database.RDB.Set(ctx, key, string(data), expiration).Err()
}

// GetUserToken 獲取 Token 對應的用戶快取資訊
func GetUserToken(token string) (*contexts.UserInfo, error) {
	ctx := context.Background()
	key := "token:" + token

	val, err := database.RDB.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var userInfo contexts.UserInfo
	err = json.Unmarshal([]byte(val), &userInfo)
	if err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// DeleteUserToken 使 Token 立即失效
func DeleteUserToken(token string) error {
	ctx := context.Background()
	key := "token:" + token
	return database.RDB.Del(ctx, key).Err()
}

// GetUserLatestLoginTime 獲取該用戶目前新登入的時間戳記
func GetUserLatestLoginTime(userID uint) (int64, error) {
	ctx := context.Background()
	key := fmt.Sprintf("user:latest_login_time:%d", userID)

	val, err := database.RDB.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	loginAt, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}

	return loginAt, nil
}

// BindUserLatestLoginTime 同時綁定時間戳記與長效 Refresh Token
func BindUserLatestLoginTime(userID uint, loginAt int64, refreshToken string, expiration time.Duration) error {
	ctx := context.Background()
	timeKey := fmt.Sprintf("user:latest_login_time:%d", userID)
	_ = database.RDB.Set(ctx, timeKey, loginAt, expiration).Err()

	refreshKey := fmt.Sprintf("user:refresh_token:%d", userID)
	return database.RDB.Set(ctx, refreshKey, refreshToken, expiration).Err()
}

// GetServerRefreshToken 獲取該用戶目前全網最新、合法的長效鑰匙
func GetServerRefreshToken(userID uint) (string, error) {
	ctx := context.Background()
	key := fmt.Sprintf("user:refresh_token:%d", userID)
	return database.RDB.Get(ctx, key).Result()
}

// DeleteUserLatestLoginTime 徹底清除該用戶的最新登入時間與 RefreshToken 紀錄
func DeleteUserLatestLoginTime(userID uint) error {
	ctx := context.Background()
	timeKey := fmt.Sprintf("user:latest_login_time:%d", userID)
	refreshKey := fmt.Sprintf("user:refresh_token:%d", userID)
	return database.RDB.Del(ctx, timeKey, refreshKey).Err()
}

// SetMemberCache 設置完整的會員快取
func SetMemberCache(schema string, member *entities.Member, expiration time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("member:cache:%s:%d", schema, member.ID)

	data, err := json.Marshal(member)
	if err != nil {
		return err
	}

	return database.RDB.Set(ctx, key, string(data), expiration).Err()
}

// GetMemberCache 獲取完整的會員快取
func GetMemberCache(schema string, userID uint) (*entities.Member, error) {
	ctx := context.Background()
	key := fmt.Sprintf("member:cache:%s:%d", schema, userID)

	val, err := database.RDB.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var member entities.Member
	err = json.Unmarshal([]byte(val), &member)
	if err != nil {
		return nil, err
	}

	return &member, nil
}

// DeleteMemberCache 刪除完整的會員快取
func DeleteMemberCache(schema string, userID uint) error {
	ctx := context.Background()
	key := fmt.Sprintf("member:cache:%s:%d", schema, userID)
	return database.RDB.Del(ctx, key).Err()
}

const luaReleaseLockScript = `
if redis.call('get', KEYS[1]) == ARGV[1] then
    return redis.call('del', KEYS[1])
else
    return 0
end
`

// AcquireDistributedLock 封裝加鎖邏輯
func AcquireDistributedLock(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	return database.RDB.SetNX(ctx, key, value, ttl).Result()
}

// ReleaseDistributedLock 封裝解鎖邏輯
func ReleaseDistributedLock(ctx context.Context, key string, value string) error {
	return database.RDB.Eval(ctx, luaReleaseLockScript, []string{key}, value).Err()
}

const optimizedLuaScriptV2 = `
    local memberBalanceKey = KEYS[1]
    -- 確保 ARGV[1] 被解析為數字型態
    local totalAmount = tonumber(ARGV[1]) or 0
    local productIdsStr = ARGV[2]
    local quantitiesStr = ARGV[3]
    local schema = ARGV[4]

    -- 驗證會員錢包快取餘額，若不存在則預設為 0
    local balanceRaw = redis.call('GET', memberBalanceKey)
    local currentBalance = tonumber(balanceRaw) or 0

    -- 餘額檢查：若快取餘額低於本次交易總額，直接攔截並返回代碼 0
    if currentBalance < totalAmount then
        return 0 
    end

    -- 解析以逗號分隔的字串品項
    local function split(str)
        local t = {}
        if str == nil or str == "" then return t end
        for item in string.gmatch(str, "[^,]+") do
            table.insert(t, tonumber(item))
        end
        return t
    end

    local productIds = split(productIdsStr)
    local reqQuantities = split(quantitiesStr)

    -- 商品庫存巡檢：遍歷所有品項快取，若有任一商品庫存不足，直接攔截並返回代碼 -1
    for i = 1, #productIds do
        local stockKey = "cache:product:stock:" .. schema .. ":" .. productIds[i]
        local currentStock = tonumber(redis.call('GET', stockKey) or '0') or 0
        if currentStock < reqQuantities[i] then
            return -1 
        end
    end

    -- 條件全數通過，執行記憶體原子扣減
    redis.call('DECRBY', memberBalanceKey, math.floor(totalAmount))
    for i = 1, #productIds do
        local stockKey = "cache:product:stock:" .. schema .. ":" .. productIds[i]
        redis.call('DECRBY', stockKey, reqQuantities[i])
    end

    return 1 
`

// DecrementTradeAssetsV2 執行前線全記憶體原子扣減
// 接收 int64 型態的總金額，並透過 strconv 轉換為純數字字串，切斷小數點引發的浮點數比對錯位漏洞。
func DecrementTradeAssetsV2(ctx context.Context, memberBalanceKey string, totalAmount int64, idList, qtyList, schema string) (int64, error) {
	return database.RDB.Eval(ctx, optimizedLuaScriptV2,
		[]string{memberBalanceKey},
		strconv.FormatInt(totalAmount, 10),
		idList,
		qtyList,
		schema,
	).Int64()
}

// CompensateTradeAssetsV2 V2版 專屬逆向還原記憶體補償機制
func CompensateTradeAssetsV2(ctx context.Context, memberBalanceKey string, totalAmount int64, idList, qtyList, schema string) {
	// 還原會員錢包餘額
	_ = database.RDB.IncrBy(ctx, memberBalanceKey, totalAmount)

	if idList == "" {
		return
	}

	// 切割字串還原商品庫存
	var ids []string
	var qtys []string

	current := ""
	for i := 0; i < len(idList); i++ {
		if idList[i] == ',' {
			ids = append(ids, current)
			current = ""
		} else {
			current += string(idList[i])
		}
	}
	if current != "" {
		ids = append(ids, current)
	}

	current = ""
	for i := 0; i < len(qtyList); i++ {
		if qtyList[i] == ',' {
			qtys = append(qtys, current)
			current = ""
		} else {
			current += string(qtyList[i])
		}
	}
	if current != "" {
		qtys = append(qtys, current)
	}

	// 巡迴還原補給回 Redis
	for i := 0; i < len(ids); i++ {
		stockKey := fmt.Sprintf("cache:product:stock:%s:%s", schema, ids[i])
		qty, _ := strconv.ParseInt(qtys[i], 10, 64)
		_ = database.RDB.IncrBy(ctx, stockKey, qty)
	}
}
