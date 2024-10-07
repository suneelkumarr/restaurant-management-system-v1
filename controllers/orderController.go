package controllers

import (
	"context"
	"net/http"
	"restorent-management/database"
	"restorent-management/models"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var orderCollection *mongo.Collection = database.OpenCollection(database.Client, "order")

func GetOrders() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		result, err := orderCollection.Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while listing order items"})
			return
		}

		var allOrders []bson.M
		if err = result.All(ctx, &allOrders); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while decoding order items"})
			return
		}

		c.JSON(http.StatusOK, allOrders)
	}
}

func GetOrderByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		orderId := c.Param("order_id")
		var order models.Order

		err := orderCollection.FindOne(ctx, bson.M{"order_id": orderId}).Decode(&order)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while fetching the order"})
			}
			return
		}

		c.JSON(http.StatusOK, order)
	}
}

func CreateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var table models.Table
		var order models.Order

		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validationErr := validate.Struct(order)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		if order.Table_id != nil {
			err := tableCollection.FindOne(ctx, bson.M{"table_id": order.Table_id}).Decode(&table)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Table was not found"})
				return
			}
		}

		orderId, err := OrderItemOrderCreator(ctx, order)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Order item was not created"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"order_id": orderId})
	}
}

func UpdateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var table models.Table
		var order models.Order
		var updateObj primitive.D

		orderId := c.Param("order_id")
		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if order.Table_id != nil {
			err := tableCollection.FindOne(ctx, bson.M{"table_id": order.Table_id}).Decode(&table)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Table was not found"})
				return
			}
			updateObj = append(updateObj, bson.E{"table_id", order.Table_id})
		}

		order.Updated_at = time.Now()
		updateObj = append(updateObj, bson.E{"updated_at", order.Updated_at})

		upsert := true
		filter := bson.M{"order_id": orderId}
		opt := options.Update().SetUpsert(upsert)

		result, err := orderCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{"$set", updateObj},
			},
			opt,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Order item update failed"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func OrderItemOrderCreator(ctx context.Context, order models.Order) (string, error) {
	order.Created_at = time.Now()
	order.Updated_at = time.Now()
	order.ID = primitive.NewObjectID()
	order.Order_id = order.ID.Hex()

	_, err := orderCollection.InsertOne(ctx, order)
	if err != nil {
		return "", err
	}

	return order.Order_id, nil
}
