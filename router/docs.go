package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/gin-gonic/gin"
)

func SetDocsRouter(router *gin.Engine) {
	docs := router.Group("/docs")
	docs.Use(func(c *gin.Context) {
		c.Set("route_tag", "docs")
	})
	{
		docs.GET("", controller.DocsHandler)
		docs.GET("/json", controller.DocsJSONHandler)
	}
}
