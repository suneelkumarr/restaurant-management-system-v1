package controllers

import (
	"context"
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

type OrderItemPack struct {
	Table_id    *string
	Order_items []models.OrderItem
}

var orderItemCollection *mongo.Collection = database.OpenCollection(database.Client, "orderItem")
var validate = validator.New()

func GetOrderItems() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		result, err := orderItemCollection.Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while listing ordered items"})
			return
		}

		var allOrderItems []bson.M
		if err = result.All(ctx, &allOrderItems); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while decoding ordered items"})
			return
		}

		c.JSON(http.StatusOK, allOrderItems)
	}
}

func GetOrderItemsByOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderId := c.Param("order_id")

		allOrderItems, err := ItemsByOrder(orderId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while listing order items by order ID"})
			return
		}

		c.JSON(http.StatusOK, allOrderItems)
	}
}

func ItemsByOrder(id string) ([]primitive.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		// Match documents with the given order_id
		{{"$match", bson.D{{"order_id", id}}}},

		// Lookup the food details
		{{"$lookup", bson.D{
			{"from", "food"},
			{"localField", "food_id"},
			{"foreignField", "food_id"},
			{"as", "food"},
		}}},

		// Lookup the order details
		{{"$lookup", bson.D{
			{"from", "order"},
			{"localField", "order_id"},
			{"foreignField", "order_id"},
			{"as", "order"},
		}}},

		// Lookup the table details
		{{"$lookup", bson.D{
			{"from", "table"},
			{"localField", "order.table_id"},
			{"foreignField", "table_id"},
			{"as", "table"},
		}}},

		// Project the necessary fields
		{{"$project", bson.D{
			{"_id", 0},
			{"amount", bson.D{{"$arrayElemAt", bson.A{"$food.price", 0}}}},
			{"food_name", bson.D{{"$arrayElemAt", bson.A{"$food.name", 0}}}},
			{"food_image", bson.D{{"$arrayElemAt", bson.A{"$food.food_image", 0}}}},
			{"table_number", bson.D{{"$arrayElemAt", bson.A{"$table.table_number", 0}}}},
			{"table_id", bson.D{{"$arrayElemAt", bson.A{"$table.table_id", 0}}}},
			{"order_id", 1},
			{"price", bson.D{{"$arrayElemAt", bson.A{"$food.price", 0}}}},
			{"quantity", 1},
		}}},

		// Group the results
		{{"$group", bson.D{
			{"_id", bson.D{
				{"order_id", "$order_id"},
				{"table_id", "$table_id"},
				{"table_number", "$table_number"},
			}},
			{"payment_due", bson.D{{"$sum", "$amount"}}},
			{"total_count", bson.D{{"$sum", 1}}},
			{"order_items", bson.D{{"$push", "$$ROOT"}}},
		}}},

		// Final projection
		{{"$project", bson.D{
			{"_id", 0},
			{"payment_due", 1},
			{"total_count", 1},
			{"table_number", "$_id.table_number"},
			{"order_items", 1},
		}}},
	}

	cursor, err := orderItemCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var OrderItems []primitive.M
	if err := cursor.All(ctx, &OrderItems); err != nil {
		return nil, err
	}

	return OrderItems, nil
}

func GetOrderItemsByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		orderItemId := c.Param("order_item_id")
		var orderItem models.OrderItem

		err := orderItemCollection.FindOne(ctx, bson.M{"order_item_id": orderItemId}).Decode(&orderItem)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusNotFound, gin.H{"error": "Order item not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while fetching the ordered item"})
			}
			return
		}

		c.JSON(http.StatusOK, orderItem)
	}
}

func UpdateOrderItems() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var orderItem models.OrderItem
		orderItemId := c.Param("order_item_id")

		if err := c.BindJSON(&orderItem); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		filter := bson.M{"order_item_id": orderItemId}
		updateObj := bson.D{}

		if orderItem.Unit_price != nil {
			updateObj = append(updateObj, bson.E{"unit_price", orderItem.Unit_price})
		}

		if orderItem.Quantity != nil {
			updateObj = append(updateObj, bson.E{"quantity", orderItem.Quantity})
		}

		if orderItem.Food_id != nil {
			updateObj = append(updateObj, bson.E{"food_id", orderItem.Food_id})
		}

		orderItem.Updated_at = time.Now()
		updateObj = append(updateObj, bson.E{"updated_at", orderItem.Updated_at})

		upsert := true
		opt := options.Update().SetUpsert(upsert)

		result, err := orderItemCollection.UpdateOne(
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

func CreateOrderItems() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var orderItemPack OrderItemPack
		var order models.Order

		if err := c.BindJSON(&orderItemPack); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validationErr := validate.Struct(orderItemPack)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		order.Order_Date = time.Now()
		order.Table_id = orderItemPack.Table_id

		orderId, err := OrderItemOrderCreator(ctx, order)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Order creation failed"})
			return
		}

		orderItemsToBeInserted := []interface{}{}
		for _, orderItem := range orderItemPack.Order_items {
			orderItem.Order_id = orderId

			validationErr := validate.Struct(orderItem)
			if validationErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
				return
			}

			orderItem.ID = primitive.NewObjectID()
			orderItem.Created_at = time.Now()
			orderItem.Updated_at = time.Now()
			orderItem.Order_item_id = orderItem.ID.Hex()

			num := toFixed(*orderItem.Unit_price, 2)
			orderItem.Unit_price = &num

			orderItemsToBeInserted = append(orderItemsToBeInserted, orderItem)
		}

		insertedOrderItems, err := orderItemCollection.InsertMany(ctx, orderItemsToBeInserted)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert order items"})
			return
		}

		c.JSON(http.StatusOK, insertedOrderItems)
	}
}
