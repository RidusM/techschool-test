// nolint: revive,staticcheck
// swagger:meta
package httpt

import "wbtest/internal/entity"

// swagger:model ErrorResponse
type ErrorResponse struct {
	Error   string `json:"error"`
}

// swagger:model SuccessResponse
type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// swagger:model Order
type Order entity.Order

// swagger:model Delivery
type Delivery entity.Delivery

// swagger:model Payment
type Payment entity.Payment

// swagger:model Item
type Item entity.Item
