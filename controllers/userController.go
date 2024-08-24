package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"restorent-management/database"
	"restorent-management/helper"
	"restorent-management/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var (
	userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")
	ErrEmailInUse                    = errors.New("this email is already in use")
	ErrPhoneInUse                    = errors.New("this phone number is already in use")
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func VerifyPassword(userPassword string, providedPassword string) (bool, string) {

	err := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	check := true
	msg := ""

	if err != nil {
		msg = fmt.Sprintf("login or password is incorrect")
		check = false
	}
	return check, msg
}

func SignUp() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var user models.User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create a new validator instance
		validate := validator.New()

		// Validate the User struct
		err := validate.Struct(user)
		if err != nil {
			// Validation failed, handle the error
			errors := err.(validator.ValidationErrors)
			c.JSON(http.StatusBadRequest, gin.H{"error": errors})
			return
		}

		// Check if email is already in use
		if count, err := userCollection.CountDocuments(ctx, bson.M{"email": user.Email}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error checking email"})
			return
		} else if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": ErrEmailInUse.Error()})
			return
		}

		// Check if phone is already in use
		if count, err := userCollection.CountDocuments(ctx, bson.M{"phone": user.Phone}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error checking phone number"})
			return
		} else if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": ErrPhoneInUse.Error()})
			return
		}

		// Hash password
		hashedPassword, err := HashPassword(user.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error hashing password"})
			return
		}
		user.Password = hashedPassword

		// Set timestamps and ID
		now := time.Now()
		user.Created_at = now
		user.Updated_at = now
		user.ID = primitive.NewObjectID()
		user.User_id = user.ID.Hex()

		// Generate tokens
		token, refreshToken, err := helper.GenerateAllTokens(user.Email, user.First_name, user.Last_name, user.User_id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error generating tokens"})
			return
		}
		user.Token = token
		user.Refresh_Token = refreshToken

		// Insert user into database
		result, err := userCollection.InsertOne(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating user"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "User created successfully", "userId": result.InsertedID, "data": result})
	}
}

func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var user models.User
		var foundUser models.User

		//convert the login data from postman which is in JSON to golang readable format
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		//find a user with that email and see if that user even exists
		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		//then you will verify the password
		passwordIsValid, msg := VerifyPassword(user.Password, foundUser.Password)

		if !passwordIsValid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
			return
		}

		//if all goes well, then you'll generate tokens
		token, refreshToken, _ := helper.GenerateAllTokens(foundUser.Email, foundUser.First_name, foundUser.Last_name, foundUser.User_id)
		helper.UpdateAllTokens(token, refreshToken, foundUser.User_id)

		//return statusOK
		c.JSON(http.StatusOK, foundUser)

	}
}

/*
*
GetOneUser fetches a specific user from the database based on the provided user ID.

@param c *gin.Context
@return gin.HandlerFunc

The function takes a gin.Context as a parameter and returns a gin.HandlerFunc. It retrieves the user ID from the URL parameter "user_id". It then uses the provided userCollection to find a user with the matching user ID. If the user is found, it returns the user in the JSON response with status code 200 (OK). If the user is not found, it returns an error message with status code 500 (Internal Server Error).

@see https://godoc.org/github.com/gin-gonic/gin
@see https://godoc.org/go.mongodb.org/mongo-driver/bson
@see https://godoc.org/go.mongodb.org/mongo-driver/mongo
@see https://godoc.org/go.mongodb.org/mongo-driver/mongo/options
*
*/
func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		userId := c.Param("user_id")
		var user models.User
		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while fetching user"})
			return
		}
		c.JSON(http.StatusOK, user)
	}
}

/*
*
GetUsers fetches users from the database and returns them along with the total count, page number, and records per page.

@param c *gin.Context
@return gin.HandlerFunc

The function takes a gin.Context as a parameter and returns a gin.HandlerFunc. It sets the default values for the record per page and page number if they are not provided in the query parameters. It then calculates the start index based on the page and record per page values.

The function uses the provided userCollection to find users with the specified projection and pagination options. It decodes the cursor into a slice of bson.M documents and counts the total number of users.

Finally, it returns a JSON response containing the users, total count, page number, and records per page.

@see https://godoc.org/github.com/gin-gonic/gin
@see https://godoc.org/go.mongodb.org/mongo-driver/bson
@see https://godoc.org/go.mongodb.org/mongo-driver/mongo
@see https://godoc.org/go.mongodb.org/mongo-driver/mongo/options
*/
func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		recordPerPage, err := strconv.Atoi(c.DefaultQuery("recordPerPage", "10"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}

		page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
		if err != nil || page < 1 {
			page = 1
		}

		startIndex := (page - 1) * recordPerPage

		matchStage := bson.D{{}}
		opts := options.Find().
			SetSkip(int64(startIndex)).
			SetLimit(int64(recordPerPage)).
			SetProjection(bson.D{
				{"password", 0},
				{"refresh_token", 0},
			})

		cursor, err := userCollection.Find(ctx, matchStage, opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while fetching users"})
			return
		}
		defer cursor.Close(ctx)

		var users []bson.M
		if err = cursor.All(ctx, &users); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while decoding user data"})
			return
		}

		totalCount, err := userCollection.CountDocuments(ctx, matchStage)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while counting total users"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"users":       users,
			"total_count": totalCount,
			"page":        page,
			"per_page":    recordPerPage,
		})
	}
}

func UpdateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var user models.User

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userId := c.Param("user_id")

		if user.Password != "" {
			hashedPassword, err := HashPassword(user.Password)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "error hashing password"})
				return
			}
			user.Password = hashedPassword
		}

		filter := bson.M{"user_id": userId}

		// Create a map to hold the fields to update
		updateFields := make(map[string]interface{})

		// Iterate through the struct fields and add non-zero values to updateFields
		val := reflect.ValueOf(user)
		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			if !field.IsZero() && typ.Field(i).Name != "ID" { // Exclude the ID field
				updateFields[typ.Field(i).Name] = field.Interface()
			}
		}
		fmt.Println(updateFields)
		update := bson.M{"$set": updateFields}

		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		_, err := userCollection.UpdateOne(ctx, filter, update, &opt)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
	}
}
