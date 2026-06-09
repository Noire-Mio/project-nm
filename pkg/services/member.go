package services

import (
	"context"
	"fmt"
	"project-nm/pkg/contexts"
	"project-nm/pkg/database"
	"project-nm/pkg/entities"
	"project-nm/pkg/utils"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

type IMemberService interface {
	GetMember(c *contexts.Member) (*entities.Member, error)
	GetMemberMQ(c *contexts.Member) (*entities.Member, error)
}

type MemberService struct{}

func (srv *MemberService) GetMember(c *contexts.Member) (*entities.Member, error) {
	MemberSchema := c.UserInfo.Schema
	id := c.UserInfo.UserID

	// 查詢資料庫確認使用者是否存在
	isExist, member, err := c.MemberRepo.Get(MemberSchema, id)
	if err != nil {
		return nil, fmt.Errorf("查詢資料庫失敗: %w", err)
	}

	// 新用戶同步寫入資料庫
	if !isExist {
		newMember := &entities.Member{
			ID:       id,
			Username: c.UserInfo.Name,
			Balance:  decimal.NewFromInt(0),
		}

		// 寫入資料庫
		err = c.MemberRepo.Create(MemberSchema, *newMember)
		if err != nil {
			return nil, fmt.Errorf("同步建立會員失敗: %w", err)
		}

		// 重新從資料庫撈取最新狀態
		_, finalMember, err := c.MemberRepo.Get(MemberSchema, id)

		if err != nil {
			return nil, fmt.Errorf("建立後二次確認失敗: %w", err)
		}
		if finalMember == nil {
			return newMember, nil
		}

		return finalMember, nil
	}

	return member, nil
}

func (srv *MemberService) GetMemberMQ(c *contexts.Member) (*entities.Member, error) {
	MemberSchema := c.UserInfo.Schema
	id := c.UserInfo.UserID
	streamName := "stream:member_init_tasks"
	ctx := context.Background()

	// 檢查完整會員資料快照
	cachedMember, err := utils.GetMemberCache(MemberSchema, id)
	if err == nil && cachedMember != nil {
		return cachedMember, nil
	}

	// 撈取既有用戶
	isExist, member, err := c.MemberRepo.Get(MemberSchema, id)
	if err != nil {
		return nil, fmt.Errorf("查詢資料庫失敗: %w", err)
	}

	memberBalanceKey := fmt.Sprintf("cache:member:balance:%s:%d", MemberSchema, id)

	// 如果舊帳號在資料庫中存在，必須同步寫入完整快照與 V2 原子錢包餘額
	if isExist {
		_ = utils.SetMemberCache(MemberSchema, member, 30*time.Minute)

		_ = database.RDB.Set(ctx, memberBalanceKey, member.Balance.String(), 0).Err()

		return member, nil
	}

	// 新註冊用戶之非同步初始化動線
	newMember := &entities.Member{
		ID:       id,
		Username: c.UserInfo.Name,
		Balance:  decimal.NewFromInt(1000),
	}

	// 寫入基礎資訊快照
	_ = utils.SetMemberCache(MemberSchema, newMember, 30*time.Minute)

	// 前線網關搶先鎖定建立 V2 原子錢包計數器
	_ = database.RDB.Set(ctx, memberBalanceKey, newMember.Balance.String(), 0).Err()

	taskMap := map[string]interface{}{
		"user_id":  strconv.FormatUint(uint64(id), 10),
		"username": c.UserInfo.Name,
		"schema":   MemberSchema,
	}

	err = utils.PushToStream(streamName, taskMap)
	if err != nil {
		_ = utils.DeleteMemberCache(MemberSchema, id)
		_ = database.RDB.Del(ctx, memberBalanceKey).Err()
		return nil, fmt.Errorf("推入非同步建立佇列失敗: %w", err)
	}

	return newMember, nil
}
