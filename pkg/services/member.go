package services

import (
	"fmt"
	"project-nm/pkg/contexts"
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

	cachedMember, err := utils.GetMemberCache(MemberSchema, id)
	if err == nil && cachedMember != nil {
		return cachedMember, nil
	}

	isExist, member, err := c.MemberRepo.Get(MemberSchema, id)
	if err != nil {
		return nil, fmt.Errorf("查詢資料庫失敗: %w", err)
	}

	if isExist {
		_ = utils.SetMemberCache(MemberSchema, member, 30*time.Minute)
		return member, nil
	}

	newMember := &entities.Member{
		ID:       id,
		Username: c.UserInfo.Name,
		Balance:  decimal.NewFromInt(0),
	}

	// 寫入快取
	_ = utils.SetMemberCache(MemberSchema, newMember, 30*time.Minute)

	taskMap := map[string]interface{}{
		"user_id":  strconv.FormatUint(uint64(id), 10),
		"username": c.UserInfo.Name,
		"schema":   MemberSchema,
	}

	err = utils.PushToStream(streamName, taskMap)
	if err != nil {
		_ = utils.DeleteMemberCache(MemberSchema, id)
		return nil, fmt.Errorf("推入非同步建立佇列失敗: %w", err)
	}

	return newMember, nil
}
