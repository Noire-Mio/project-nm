package dtos

type TradeDto struct {
	ProductID uint   `json:"product_id"`
	Quantity  int64  `json:"quantity"`
	Type      string `json:"type"` // "pickup" (買入) 或 "return" (退貨)
}
