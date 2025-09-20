package entity

import "github.com/google/uuid"

type Item struct {
	ChrtID      uint64    `json:"chrt_id"      validate:"required,gte=1"`
	TrackNumber string    `json:"track_number" validate:"required,max=50"`
	Price       uint64    `json:"price"        validate:"required,gte=1"`
	Rid         uuid.UUID `json:"rid"          validate:"required,uuid_strict"`
	Name        string    `json:"name"         validate:"required,max=255"`
	Sale        int       `json:"sale"         validate:"gte=0,lte=100"`
	Size        string    `json:"size"         validate:"required"`
	TotalPrice  uint64    `json:"total_price"  validate:"required,gte=1"`
	NMID        uint64    `json:"nm_id"        validate:"required,gte=1"`
	Brand       string    `json:"brand"        validate:"required,max=100"`
	Status      int       `json:"status"       validate:"gte=0"`
}
