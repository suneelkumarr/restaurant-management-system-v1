package controllers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"restorent-management/database"
	"restorent-management/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

var foodCollection *mongo.Collection = database.OpenCollection(database.Client, "food")

func CreateFood() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		var menu models.Menu
		var food models.Food

		if err := c.ShouldBindJSON(&food); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create a new validator instance
		validate := validator.New()

		// Validate the food struct
		err := validate.Struct(food)

		if err != nil {
			// Validation failed, handle the error
			errors := err.(validator.ValidationErrors)
			c.JSON(http.StatusBadRequest, gin.H{"error": errors})
			return
		}

		menudata := menuCollection.FindOne(ctx, bson.M{"menu_id": food.Menu_id}).Decode(&menu)
		defer cancel()

		if menudata != nil {
			msg := fmt.Sprintf("menu was not found")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		// Set timestamps and ID
		now := time.Now()
		food.Created_at = now
		food.Updated_at = now
		food.ID = primitive.NewObjectID()
		food.Food_id = food.ID.Hex()
		var num = toFixed(*food.Price, 2)
		food.Price = &num
		result, insertErr := foodCollection.InsertOne(ctx, food)

		if insertErr != nil {
			msg := insertErr.Error()
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		defer cancel()
		c.JSON(http.StatusOK, result)
	}

}

func GetFoods() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		page, err := strconv.Atoi(c.Query("page"))
		if err != nil || page < 1 {
			page = 1
		}

		limit, err := strconv.Atoi(c.Query("limit"))
		if err != nil || limit < 1 || limit > 100 {
			limit = 10
		}

		foodFilter := bson.M{}

		foodCollection := database.OpenCollection(database.Client, "food")

		foodCursor, err := foodCollection.Find(ctx, foodFilter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer foodCursor.Close(ctx)
		var foodList []bson.M
		for foodCursor.Next(ctx) {
			var food bson.M
			err := foodCursor.Decode(&food)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			foodList = append(foodList, food)
		}
		defer cancel()
		c.JSON(http.StatusOK, foodList)
	}
}

func GetFoodByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		foodId := c.Param("food_id")
		var food models.Menu

		err := foodCollection.FindOne(ctx, bson.M{"food_id": foodId}).Decode(&food)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while fetching the menu"})
		}
		c.JSON(http.StatusOK, food)
	}
}

func UpdateFood() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var food models.Food
		foodID := c.Param("food_id")

		if err := c.ShouldBindJSON(&food); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updateObj := bson.M{}

		if food.Name != nil {
			updateObj["name"] = *food.Name
		}

		if food.Price != nil {
			updateObj["price"] = *food.Price
		}

		if food.Food_image != nil {
			updateObj["food_image"] = *food.Food_image
		}

		if food.Menu_id != nil {
			var menu models.Menu
			err := menuCollection.FindOne(ctx, bson.M{"menu_id": *food.Menu_id}).Decode(&menu)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Menu not found"})
				return
			}
			updateObj["menu_id"] = *food.Menu_id
		}

		updateObj["updated_at"] = time.Now()

		filter := bson.M{"food_id": foodID}
		update := bson.M{"$set": updateObj}

		opts := options.Update().SetUpsert(true)

		result, err := foodCollection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Food item update failed"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
