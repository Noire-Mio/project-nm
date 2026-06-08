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
	clients "project-nm/pkg/grpc/client"
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
	Price     decimal.Decimal
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

	// 分散式鎖防禦機制
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

	// 透過倉庫執行悲觀鎖定並撈取商品最新狀態
	dbProducts, err := c.TradeRepo.GetProductsByIDsForUpdate(schema, productIDs)
	if err != nil {
		return nil, fmt.Errorf("PRODUCT_DB_ERROR: 透過倉庫悲觀鎖定並撈取商品資料失敗: %w", err)
	}

	productMap := make(map[uint]entities.Product)
	for _, p := range dbProducts {
		productMap[p.ID] = p
	}

	// 透過倉庫執行悲觀鎖定並撈取會員最新餘額
	exists, memberEntity, err := c.MemberRepo.GetForUpdate(schema, userID)
	if err != nil {
		return nil, fmt.Errorf("MEMBER_DB_SYSTEM_ERROR: 透過倉庫悲觀鎖定並讀取會員資產系統異常: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("MEMBER_NOT_FOUND: 資料庫中不存在此會員 (UserID: %d)", userID)
	}

	var totalAmount decimal.Decimal
	tasks := make([]validatedTask, 0, len(dtos))

	for _, dto := range dtos {
		product, exists := productMap[dto.ProductID]
		if !exists {
			return nil, fmt.Errorf("PRODUCT_NOT_FOUND: 商品 ID %d 不存在於系統中", dto.ProductID)
		}

		if dto.Type == "pickup" && product.Stock < dto.Quantity {
			return nil, fmt.Errorf("PRODUCT_OUT_OF_STOCK: 商品 ID %d 在資料庫中庫存不足", dto.ProductID)
		}

		itemTotalCost := product.Price.Mul(decimal.NewFromInt(dto.Quantity))

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
			Price:     product.Price,
			TotalCost: itemTotalCost,
			TxType:    dto.Type,
		})
	}

	// 會員錢包餘額邊界審查
	memberEntity.Balance = memberEntity.Balance.Sub(totalAmount)
	if memberEntity.Balance.IsNegative() {
		return nil, errors.New("INSUFFICIENT_BALANCE: 總餘額不足以支付購物車商品，交易拒絕")
	}

	// 同步全新資產數據至唯讀快取
	err = utils.SetMemberCache(schema, memberEntity, 30*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("CACHE_WRITE_FAILED: 更新會員快取失敗: %w", err)
	}

	// 將審查通過的任務包裝推入消息佇列解耦
	mqPipe := database.RDB.Pipeline()
	for _, task := range tasks {
		taskMap := map[string]interface{}{
			"user_id":    strconv.FormatUint(uint64(userID), 10),
			"product_id": strconv.FormatUint(uint64(task.ProductID), 10),
			"quantity":   strconv.FormatInt(task.Quantity, 10),
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
		return nil, fmt.Errorf("MQ_PUSH_FAILED: 推入批量交易佇列失敗: %w", err)
	}

	pendingTx := &entities.Transaction{
		MemberID: userID,
		Status:   "pending",
	}

	return pendingTx, nil
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

	grpcClient, err := clients.NewProjectNMGrpcClient()
	if err != nil {
		return nil, fmt.Errorf("GRPC_CLIENT_INIT_FAILED: 建立 gRPC 客戶端失敗: %w", err)
	}

	grpcResp, err := grpcClient.ExecuteOrder(grpcUserInfo, grpcItems)
	if err != nil {
		return nil, fmt.Errorf("REMOTE_EXECUTE_FAILED: 遠端核心交易執行失敗: %w", err)
	}
	return grpcResp, nil
}
