// swagger:meta
package httpt

import "wbtest/internal/entity"

// swagger:model ErrorResponse
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// swagger:model SuccessResponse
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// swagger:model Order
type Order entity.Order

// swagger:model Delivery
type Delivery entity.Delivery

// swagger:model Payment
type Payment entity.Payment

// swagger:model Item
type Item entity.Item
