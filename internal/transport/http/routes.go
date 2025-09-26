package httpt

import (
	"net/http"

	_ "wbtest/docs" // for swagger

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Order Service API
// @version         1.0
// @description     API для управления заказами
// @termsOfService  http://swagger.io/terms/
// @contact.name    API Support
// @contact.email   support@example.com
// @license.name    Apache 2.0
// @license.url     http://www.apache.org/licenses/LICENSE-2.0.html
// @host            localhost:8080
// @BasePath        /api/v1
// @schemes         http https
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
func (h *OrderHandler) setupRoutes() {
	h.router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	h.router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})

	orders := h.router.Group("/orders")
	{
		orders.GET("/:order_uid", h.getOrderHandler)
	}

	h.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
