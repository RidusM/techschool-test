package httpt

import (
	"wbtest/internal/service"
	"wbtest/pkg/logger"
	"wbtest/pkg/metric"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	svc     *service.OrderService
	log     logger.Logger
	metrics metric.HTTP
	router  *gin.Engine
}

func NewOrderHandler(
	svc *service.OrderService,
	log logger.Logger,
	metrics metric.HTTP,
) *OrderHandler {
	h := &OrderHandler{
		svc:     svc,
		log:     log,
		metrics: metrics,
	}

	router := gin.New()

	router.Use(h.requestIDMiddleware())
	router.Use(h.loggingMiddleware())
	router.Use(gin.Recovery())

	h.router = router
	h.setupRoutes()

	return h
}

func (h *OrderHandler) Engine() *gin.Engine {
	h.router.LoadHTMLGlob("web/*.html")
	return h.router
}
