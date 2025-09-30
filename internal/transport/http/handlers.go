package httpt

import (
	"context"
	"net/http"
	"time"

	"wbtest/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	_defaultContextTimeout = 500 * time.Millisecond
)

// @Summary Получить заказ
// @Description Возвращает заказ по уникальному идентификатору
// @Tags Orders
// @Accept json
// @Produce json
// @Param order_uid path string true "Уникальный идентификатор заказа"
// @Success 200 {object} entity.Order "Успешный ответ с данными заказа"
// @Failure 400 {object} httpt.ErrorResponse "Неверный формат order_uid"
// @Failure 404 {object} httpt.ErrorResponse "Заказ не найден"
// @Failure 500 {object} httpt.ErrorResponse "Внутренняя ошибка сервера"
// @Router /orders/{order_uid} [get]
func (h *OrderHandler) getOrderHandler(c *gin.Context) {
	const op = "transport.getOrderHandler"

	log := h.log.Ctx(c.Request.Context())
	orderUIDStr := c.Param("order_uid")

	orderUID, err := uuid.Parse(orderUIDStr)
	if err != nil {
		h.handleInvalidUUID(c, op, orderUIDStr)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), _defaultContextTimeout)
	defer cancel()

	order, err := h.svc.GetOrder(ctx, orderUID)
	if err != nil {
		h.handleServiceError(c, err, op)
		return
	}

	log.LogAttrs(ctx, logger.InfoLevel, "order retrieved successfully",
		logger.String("order_uid", orderUIDStr),
	)

	c.JSON(http.StatusOK, order)
}
