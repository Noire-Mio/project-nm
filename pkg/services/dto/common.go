package services

import (
	"fmt"
	"project-nm/pkg/contexts"
)

// **************************************
// 常用功能函數。
// **************************************
func getOrderBy(columns []string, booleans []bool) (orderBy []string) {
	for i, column := range columns {
		if booleans == nil || i >= len(booleans) || !booleans[i] {
			orderBy = append(orderBy, fmt.Sprintf("%s %s", column, "asc"))
		} else {
			orderBy = append(orderBy, fmt.Sprintf("%s %s", column, "desc"))
		}
	}
	return
}

func GetOrderBy(columns []string, booleans []bool) []string { return getOrderBy(columns, booleans) }

// 開始交易
func StartTransaction(c contexts.IContext, fn func() error) (err error) {
	err = c.Begin()
	defer func() {
		if err == nil {
			err = c.Commit()
		} else {
			c.Rollback()
		}
	}()
	if err != nil {
		return
	}
	err = fn()
	return
}

type DataParam struct {
	Order string `json:"order"` // 排序
	Desc  bool   `json:"desc"`  // 是否降幕
}
