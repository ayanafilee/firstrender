package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var collection *mongo.Collection

// Struct for students
type Student struct {
	Name string `json:"name" bson:"name"`
	Age  int    `json:"age"  bson:"age"`
}

func main() {
	// Load .env file for local dev (Render will skip this)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using Render environment variables")
	}

	fmt.Println("Main package -> PORT:", os.Getenv("PORT"))

	// MongoDB URI
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("You must set MONGODB_URI environment variable")
	}

	// MongoDB client
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatal("MongoDB ping failed:", err)
	}

	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")

	// Database & collection
	db := client.Database("students")
	collection = db.Collection("theirdata")

	// Gin router
	r := gin.Default()

	// CORS (allow localhost for dev + your Render domain in production)
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "https://your-render-service.onrender.com"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowCredentials: true,
	}))

	// GET /students
	r.GET("/students", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := collection.Find(ctx, bson.D{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch documents"})
			return
		}
		defer cursor.Close(ctx)

		var results []bson.M
		if err := cursor.All(ctx, &results); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode documents"})
			return
		}

		c.JSON(http.StatusOK, results)
	})

	// POST /students
	r.POST("/students", func(c *gin.Context) {
		var newStudent Student
		if err := c.ShouldBindJSON(&newStudent); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := collection.InsertOne(ctx, newStudent)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert document"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":    "Student added successfully!",
			"insertedID": result.InsertedID,
		})
	})

	// ✅ Run on Render-provided PORT
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // local fallback
	}
	r.Run(":" + port)
}
