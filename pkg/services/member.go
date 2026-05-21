package services

import (
	"fmt"
	"project-nm/pkg/contexts"
	"project-nm/pkg/entities"
	"project-nm/pkg/utils"
	"time"

	"github.com/shopspring/decimal"
)

type IMemberService interface {
	GetMember(c *contexts.Member) (*entities.Member, error)
}

type MemberService struct{}


func (srv *MemberService) GetMember(c *contexts.Member) (*entities.Member, error) {
	MemberSchema := c.UserInfo.Schema
	id := c.UserInfo.UserID
	initKey := fmt.Sprintf("tenant:member_init_flag:%s:%d", MemberSchema, id)

	// 1. 依然先用快取擋住老用戶 (QPS 2000 刷進來，老用戶直接走這回傳)
	_, err := utils.GetCache(initKey)
	if err == nil {
		_, member, _ := c.MemberRepo.Get(MemberSchema, id)
		return member, nil
	}


	// 2. 查資料庫，如果真的沒資料，觸發 MQ 削峰
	isExist, member, err := c.MemberRepo.Get(MemberSchema, id)
	if err != nil {
		return nil, err
	}

	if !isExist {
		// 🎯 3. 核心亮點：不直接塞資料庫！改為組裝任務打包丟進 MQ
		taskData := map[string]interface{}{
			"user_id":  id,
			"username": c.UserInfo.Name,
			"schema":   MemberSchema,
			
		}

		// 呼叫你 utils 寫好的 PushToStream
		streamName := "stream:member_init_tasks"
		err = utils.PushToStream(streamName, taskData)
		if err != nil {
			return nil, fmt.Errorf("系統繁忙，排隊失敗: %w", err)
		}

		// 🎯 4. 為了不讓前端拿到空資料，在記憶體直接組裝一個「預期中的會員」先回傳
		// 前端畫面上會立刻顯示，而背後的資料庫此時正在排隊寫入
		virtualMember := &entities.Member{
			ID:       id,
			Username: c.UserInfo.Name,
			Balance:  decimal.NewFromInt(0),
		}

		// 順手把快取標記補上，防止這個使用者下一微秒重複發送排隊任務
		_ = utils.SetCache(initKey, "1", 24*time.Hour)

		return virtualMember, nil
	}

	return member, nil
}
