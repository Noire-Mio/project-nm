package inputmodels

type TradeInput struct {
	ProductID uint   `json:"product_id"`
	Quantity  int64  `json:"quantity"`
	Type      string `json:"type"`
}
