package main

import (
	"context"
	"fmt"
	"log"
	"net/http" // Import the net/http package
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	// "myapp/utils" // I've commented this out as I don't have the code for it
)

var collection *mongo.Collection

// --- NEW ---
// Define a struct to map the incoming JSON to.
// This ensures our data is structured correctly.
type Student struct {
	Name string `json:"name" bson:"name"`
	Age  int    `json:"age"  bson:"age"`
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	fmt.Println("Main package -> PORT:", os.Getenv("PORT"))

	// Call utils package function
	// utils.PrintEnv()

	// Get MongoDB URI from .env
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("You must set MONGODB_URI environment variable")
	}

	// Set Stable API version for MongoDB
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Ping the MongoDB server
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatal("MongoDB ping failed:", err)
	}

	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")

	// Pick your database and collection
	db := client.Database("students")
	collection = db.Collection("theirdata")

	// Set up Gin router
	r := gin.Default()

	// Enable CORS for React dev server
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowCredentials: true,
	}))

	// Route: GET /students → fetch all documents
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

	// --- NEW ROUTE ---
	// Route: POST /students → add a new student
	r.POST("/students", func(c *gin.Context) {
		var newStudent Student

		// Bind the incoming JSON from the request body to the newStudent struct.
		// If there's an error (e.g., bad format), return a 400 Bad Request error.
		if err := c.ShouldBindJSON(&newStudent); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create a context for the database operation
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Insert the new student document into the collection
		result, err := collection.InsertOne(ctx, newStudent)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert document"})
			return
		}

		// If successful, return a 201 Created status and the ID of the new document.
		c.JSON(http.StatusCreated, gin.H{
			"message":    "Student added successfully!",
			"insertedID": result.InsertedID,
		})
	})

	// Run the server on port 8080
	r.Run(":8080")
}