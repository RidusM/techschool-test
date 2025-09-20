package entity

import (
	"github.com/google/uuid"
)

type Order struct {
	OrderUID          uuid.UUID `json:"order_uid"          validate:"required,uuid_strict"`
	TrackNumber       string    `json:"track_number"       validate:"required,max=50"`
	Entry             string    `json:"entry"              validate:"required,max=10"`
	Delivery          *Delivery `json:"delivery"           validate:"required"`
	Payment           *Payment  `json:"payment"            validate:"required"`
	Items             []*Item   `json:"items"              validate:"required,min=1,dive"`
	Locale            string    `json:"locale"             validate:"required,len=2"`
	InternalSignature string    `json:"internal_signature" validate:"max=255"`
	CustomerID        string    `json:"customer_id"        validate:"required,max=50"`
	DeliveryService   string    `json:"delivery_service"   validate:"required,max=50"`
	Shardkey          string    `json:"shardkey"           validate:"required,max=10"`
	SmID              int       `json:"sm_id"              validate:"required,gte=0"`
	DateCreated       string    `json:"date_created"       validate:"required"`
	OofShard          string    `json:"oof_shard"          validate:"required,len=1"`
}
