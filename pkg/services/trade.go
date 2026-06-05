package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"project-nm/pkg/contexts"
	"project-nm/pkg/database"
	"project-nm/pkg/entities"
	"project-nm/pkg/grpc/pb"
	"project-nm/pkg/services/dtos"
	"project-nm/pkg/utils"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

type ITradeService interface {
	ExecuteOrder(c *contexts.Trade, dtos []dtos.TradeDto) (*entities.Transaction, error)
	ProcessOrder(ctx *contexts.Trade, dtos []dtos.TradeDto) (*pb.TradeGrpcResponse, error)
}
type TradeService struct{}

type validatedTask struct {
	ProductID uint
	Quantity  int64
	RealPrice decimal.Decimal
	TotalCost decimal.Decimal
	TxType    string
}

func (srv *TradeService) ExecuteOrder(c *contexts.Trade, dtos []dtos.TradeDto) (*entities.Transaction, error) {
	if len(dtos) == 0 {
		return nil, errors.New("INVALID_REQUEST: 無任何商品")
	}

	schema := c.UserInfo.Schema
	userID := c.UserInfo.UserID
	streamName := "stream:trade_tasks"
	ctx := context.Background()

	productIDs := make([]uint, 0, len(dtos))
	for _, dto := range dtos {
		if dto.Quantity <= 0 {
			return nil, fmt.Errorf("INVALID_QUANTITY: 商品 ID %d 的數量必須大於 0", dto.ProductID)
		}
		productIDs = append(productIDs, dto.ProductID)
	}

	// 分散式鎖
	lockKey := fmt.Sprintf("lock:trade:%s:%d", schema, userID)
	lockValue := strconv.FormatInt(time.Now().UnixNano(), 10)
	lockTTL := 5 * time.Second

	acquired, err := utils.AcquireDistributedLock(ctx, lockKey, lockValue, lockTTL)
	if err != nil {
		return nil, fmt.Errorf("DISTRIBUTED_LOCK_SYSTEM_ERROR: 分散式鎖系統異常: %w", err)
	}
	if !acquired {
		return nil, errors.New("TOO_MANY_REQUESTS: 您的操作過於頻繁，請稍後再試")
	}

	defer func() {
		_ = utils.ReleaseDistributedLock(ctx, lockKey, lockValue)
	}()

	// 讀取價格
	dbProducts, err := c.TradeRepo.GetProductsByIDs(database.DB, productIDs)
	if err != nil {
		return nil, fmt.Errorf("PRODUCT_DB_ERROR: 透過倉庫撈取商品真實價格失敗: %w", err)
	}

	// 將撈出來的商品做成 Map
	productPriceMap := make(map[uint]decimal.Decimal)
	for _, p := range dbProducts {
		productPriceMap[p.ID] = p.Price
	}

	// 從 Redis 讀取會員物件
	memberCache, err := utils.GetMemberCache(schema, userID)
	if err != nil {
		return nil, fmt.Errorf("CACHE_READ_FAILED: 獲取會員記憶體資料失敗，請重新載入帳號: %w", err)
	}

	originalBalance := memberCache.Balance

	var totalAmount decimal.Decimal

	tasks := make([]validatedTask, 0, len(dtos))

	for _, dto := range dtos {
		realPrice, exists := productPriceMap[dto.ProductID]
		if !exists {
			return nil, fmt.Errorf("PRODUCT_NOT_FOUND: 商品 ID %d 不存在於系統中", dto.ProductID)
		}

		itemTotalCost := realPrice.Mul(decimal.NewFromInt(dto.Quantity))

		if dto.Type == "pickup" {
			totalAmount = totalAmount.Add(itemTotalCost)
		} else if dto.Type == "return" {
			totalAmount = totalAmount.Sub(itemTotalCost)
		} else {
			return nil, fmt.Errorf("INVALID_TX_TYPE: 商品 ID %d 帶有未知的交易類型: %s", dto.ProductID, dto.Type)
		}

		tasks = append(tasks, validatedTask{
			ProductID: dto.ProductID,
			Quantity:  dto.Quantity,
			RealPrice: realPrice,
			TotalCost: itemTotalCost,
			TxType:    dto.Type,
		})
	}

	// 餘額邊界審查
	memberCache.Balance = memberCache.Balance.Sub(totalAmount)
	if memberCache.Balance.IsNegative() {
		return nil, errors.New("INSUFFICIENT_BALANCE: 總餘額不足以支付購物車商品，交易拒絕")
	}

	// Redis Pipeline 預扣庫存
	pipe := database.RDB.Pipeline()
	cmds := make(map[uint]*redis.IntCmd)

	for _, task := range tasks {
		productStockKey := fmt.Sprintf("product:stock:%d", task.ProductID)
		if task.TxType == "pickup" {
			cmds[task.ProductID] = pipe.DecrBy(ctx, productStockKey, task.Quantity)
		} else if task.TxType == "return" {
			cmds[task.ProductID] = pipe.IncrBy(ctx, productStockKey, task.Quantity)
		}
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		srv.compensateRedisStocks(ctx, dtos)
		return nil, fmt.Errorf("REDIS_PIPELINE_ERROR: 批量預扣庫存系統異常: %w", err)
	}

	// 超賣判定
	for productID, cmd := range cmds {
		currentStock := cmd.Val()
		if currentStock < 0 {
			srv.compensateRedisStocks(ctx, dtos)
			return nil, fmt.Errorf("OUT_OF_STOCK: 商品 ID %d 庫存不足，全結帳回滾", productID)
		}
	}

	// 覆蓋會員全新餘額
	err = utils.SetMemberCache(schema, memberCache, 30*time.Minute)
	if err != nil {
		srv.compensateRedisStocks(ctx, dtos)
		return nil, fmt.Errorf("CACHE_WRITE_FAILED: 更新會員批量快取失敗，變更已撤回: %w", err)
	}

	// 寫入 MQ
	mqPipe := database.RDB.Pipeline()
	for _, task := range tasks {
		taskMap := map[string]interface{}{
			"user_id":    strconv.FormatUint(uint64(userID), 10),
			"product_id": strconv.FormatUint(uint64(task.ProductID), 10),
			"quantity":   strconv.FormatInt(task.Quantity, 10),
			"price":      task.RealPrice.String(),
			"amount":     task.TotalCost.String(),
			"type":       task.TxType,
			"schema":     schema,
		}
		mqPipe.XAdd(ctx, &redis.XAddArgs{
			Stream: streamName,
			Values: taskMap,
		})
	}

	_, err = mqPipe.Exec(ctx)
	if err != nil {
		srv.compensateRedisStocks(ctx, dtos)

		memberCache.Balance = originalBalance
		_ = utils.SetMemberCache(schema, memberCache, 30*time.Minute)

		return nil, fmt.Errorf("MQ_PUSH_FAILED: 推入批量交易佇列失敗，所有扣款與庫存變更已還原: %w", err)
	}

	pendingTx := &entities.Transaction{
		MemberID: userID,
		Status:   "pending",
	}

	return pendingTx, nil
}

func (srv *TradeService) compensateRedisStocks(ctx context.Context, dtos []dtos.TradeDto) {
	pipe := database.RDB.Pipeline()
	for _, dto := range dtos {
		productStockKey := fmt.Sprintf("product:stock:%d", dto.ProductID)
		if dto.Type == "pickup" {
			pipe.IncrBy(ctx, productStockKey, dto.Quantity)
		} else if dto.Type == "return" {
			pipe.DecrBy(ctx, productStockKey, dto.Quantity)
		}
	}
	_, _ = pipe.Exec(ctx)
}

func (srv *TradeService) ProcessOrder(ctx *contexts.Trade, dtos []dtos.TradeDto) (*pb.TradeGrpcResponse, error) {
	grpcItems := make([]*pb.TradeGrpcItem, 0, len(dtos))
	for _, d := range dtos {
		grpcItems = append(grpcItems, &pb.TradeGrpcItem{
			ProductId: uint32(d.ProductID),
			Quantity:  d.Quantity,
			Type:      d.Type,
		})
	}

	userInfo := ctx.UserInfo
	grpcUserInfo := &pb.GRPCUserInfo{
		UserId:   uint64(userInfo.UserID),
		Identity: userInfo.Identity,
		Name:     userInfo.Name,
		Schema:   userInfo.Schema,
	}

	grpcResp, err := ctx.ProjectNMGrpcClient.ExecuteOrder(grpcUserInfo, grpcItems)
	if err != nil {

		return nil, fmt.Errorf("REMOTE_EXECUTE_FAILED: 遠端核心交易執行失敗: %w", err)
	}

	return grpcResp, nil
}
