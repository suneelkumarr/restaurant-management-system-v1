package routes

import (
	"restorent-management/controllers"

	"github.com/gin-gonic/gin"
)

func InvoiceRoutes(router *gin.Engine) {
	invoiceGroup := router.Group("/invoices")
	{
		invoiceGroup.GET("", controllers.GetInvoices())
		invoiceGroup.GET("/:invoice_id", controllers.GetInvoiceByID())
		invoiceGroup.POST("/invoices", controllers.CreateInvoice())
		invoiceGroup.PATCH("/:invoice_id", controllers.UpdateInvoice())
	}
}
