package controllers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"restorent-management/database"
	"restorent-management/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var invoiceCollection *mongo.Collection = database.OpenCollection(database.Client, "invoice")

type InvoiceViewFormat struct {
	Invoice_id       string
	Payment_method   string
	Order_id         string
	Payment_status   *string
	Payment_due      interface{}
	Table_number     interface{}
	Payment_due_date time.Time
	Order_details    interface{}
}

func CreateInvoice() gin.HandlerFunc {
	// Create a validator instance outside the handler to avoid re-creating it each time
	validate := validator.New()

	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var invoice models.Invoice
		var order models.Order
		var err error

		// Bind the JSON payload to the invoice struct
		if err = c.ShouldBindJSON(&invoice); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Set default payment status if not provided
		if invoice.Payment_status == nil {
			status := "PENDING"
			invoice.Payment_status = &status
		}

		// Validate the invoice struct
		if err = validate.Struct(invoice); err != nil {
			// Extract and return validation errors
			var validationErrors []string
			for _, err := range err.(validator.ValidationErrors) {
				validationErrors = append(validationErrors, fmt.Sprintf("Field %s: %s", err.Field(), err.ActualTag()))
			}
			c.JSON(http.StatusBadRequest, gin.H{"errors": validationErrors})
			return
		}

		// Check if the associated order exists
		err = orderCollection.FindOne(ctx, bson.M{"order_id": invoice.Order_id}).Decode(&order)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			} else {
				log.Printf("Failed to find order: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			}
			return
		}

		// Set timestamps and IDs
		now := time.Now()
		invoice.Payment_due_date = now.AddDate(0, 0, 1)
		invoice.Created_at = now
		invoice.Updated_at = now
		invoice.ID = primitive.NewObjectID()
		invoice.Invoice_id = invoice.ID.Hex()

		// Insert the invoice into the database
		result, insertErr := invoiceCollection.InsertOne(ctx, invoice)
		if insertErr != nil {
			log.Printf("Failed to insert invoice: %v", insertErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func GetInvoices() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := invoiceCollection.Find(ctx, bson.D{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var invoice []models.Invoice
		if err := cursor.All(ctx, &invoice); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, invoice)
	}
}

func GetInvoiceByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		InvoiceId := c.Param("invoice_id")
		var invoice models.Invoice

		err := invoiceCollection.FindOne(ctx, bson.M{"invoice_id": InvoiceId}).Decode(&invoice)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while fetching the menu"})
		}
		var invoiceView InvoiceViewFormat

		allOrderItems, err := ItemsByOrder(invoice.Order_id)
		invoiceView.Order_id = invoice.Order_id
		invoiceView.Payment_due_date = invoice.Payment_due_date

		invoiceView.Payment_method = "null"
		if invoice.Payment_method != nil {
			invoiceView.Payment_method = *invoice.Payment_method
		}

		invoiceView.Invoice_id = invoice.Invoice_id
		invoiceView.Payment_status = invoice.Payment_status
		invoiceView.Payment_due = allOrderItems[0]["payment_due"]
		invoiceView.Table_number = allOrderItems[0]["table_number"]
		invoiceView.Order_details = allOrderItems[0]["order_items"]

		c.JSON(http.StatusOK, invoiceView)
	}
}

func UpdateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a context with a timeout to prevent hanging requests
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var invoice models.Invoice
		invoiceId := c.Param("invoice_id")

		// Bind the JSON body to the invoice struct
		if err := c.BindJSON(&invoice); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Set default payment status if it's nil
		if invoice.Payment_status == nil {
			status := "PENDING"
			invoice.Payment_status = &status
		}

		// Prepare the update object with only the fields that are not nil
		updateObj := bson.M{}
		if invoice.Payment_method != nil {
			updateObj["payment_method"] = invoice.Payment_method
		}
		if invoice.Payment_status != nil {
			updateObj["payment_status"] = invoice.Payment_status
		}

		// Update the 'updated_at' field to the current time
		invoice.Updated_at = time.Now().UTC()
		updateObj["updated_at"] = invoice.Updated_at

		// Build the filter to find the invoice by ID
		filter := bson.M{"invoice_id": invoiceId}

		// Set upsert option to true to create a new document if one doesn't exist
		opt := options.Update().SetUpsert(true)

		// Attempt to update the invoice in the collection
		result, err := invoiceCollection.UpdateOne(
			ctx,
			filter,
			bson.M{"$set": updateObj},
			opt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invoice update failed"})
			return
		}

		// Return the result of the update operation
		c.JSON(http.StatusOK, result)
	}
}
