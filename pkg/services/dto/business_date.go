package services

import (
	"errors"
	"project-nm/pkg/contexts"
)

type IBusinessDateService interface {
	GetBusinessDate(c *contexts.BusinessDate) (string, error)
}

type BusinessDateService struct{}

func (srv *BusinessDateService) GetBusinessDate(c *contexts.BusinessDate) (string, error) {
	businessDateSchema := c.UserInfo.Schema
	current, err := c.BusinessDateRepo.GetBusinessDate(businessDateSchema)
	if err != nil {
		return "", err
	}
	if current == nil {
		return "", errors.New("目前沒有營業日")
	}

	return current.BusinessDate, err
}
