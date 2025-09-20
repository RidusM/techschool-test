package entity

import "github.com/google/uuid"

type Payment struct {
	Transaction  uuid.UUID `json:"transaction"   validate:"required,uuid_strict"`
	RequestID    uuid.UUID `json:"request_id"    validate:"max=50"`
	Currency     string    `json:"currency"      validate:"required,len=3"`
	Provider     string    `json:"provider"      validate:"required,max=50"`
	Amount       uint64    `json:"amount"        validate:"required,gte=1"`
	PaymentDt    int64     `json:"payment_dt"    validate:"required,unix_timestamp"`
	Bank         string    `json:"bank"          validate:"required,max=50"`
	DeliveryCost uint64    `json:"delivery_cost" validate:"required,gte=0"`
	GoodsTotal   uint64    `json:"goods_total"   validate:"required,gte=1"`
	CustomFee    uint64    `json:"custom_fee"    validate:"gte=0"`
}
