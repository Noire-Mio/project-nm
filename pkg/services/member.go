package services

import (
	"project-nm/pkg/contexts"
	"project-nm/pkg/entities"

	"github.com/shopspring/decimal"
)

type IMemberService interface {
	GetMember(c *contexts.Member) (*entities.Member, error)
}

type MemberService struct{}

func (srv *MemberService) GetMember(c *contexts.Member) (*entities.Member, error) {
	
	MemberSchema := c.UserInfo.Schema
	id := c.UserInfo.UserID
	
	isExist, member, err := c.MemberRepo.Get(MemberSchema, id)
	if err != nil {
		return nil, err
	}
	if !isExist {
		newMember := entities.Member{
			ID:       id,
			Username: c.UserInfo.Name,
			Balance:  decimal.NewFromInt(0),
		}

		member, err = c.MemberRepo.Create(MemberSchema, newMember)
		if err != nil {
			return nil, err
		}
	}

	return member, nil
}
