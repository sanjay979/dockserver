package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
	_ "github.com/go-sql-driver/mysql"
)

// Database configuration
const (
	dbUser     = "root"
	dbPassword = ""
	dbName     = "testdb"
	dbHost     = "127.0.0.1:3306" // Default MySQL port
)

var db *sql.DB

type Document struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Application struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	Documents []Document `json:"documents"`
}

func initDB() *sql.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbName)

	// Open database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Database is unreachable: %v", err)
	}

	fmt.Println("Connected to MySQL database successfully!")
	return db
}

func main() {
	db = initDB()
	defer db.Close()

	r := gin.Default()

	// CORS Middleware
	r.Use(cors.Default()) // This will allow all origins; configure as per your requirement

// Fetch all applications for the specified user_id
r.GET("/applications", func(c *gin.Context) {
	userID := c.DefaultQuery("user_id", "0") // Get user_id from query param, default to "0" if not found
	var applications []Application
	rows, err := db.Query("SELECT id, name FROM applications WHERE user_id = ?", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch applications"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var app Application
		if err := rows.Scan(&app.ID, &app.Name); err != nil {
			log.Printf("Error scanning application: %v", err)
			continue
		}

		// Fetch documents for the application
		docRows, err := db.Query("SELECT id, name FROM documents WHERE application_id = ?", app.ID)
		if err != nil {
			log.Printf("Error fetching documents for application %d: %v", app.ID, err)
			continue
		}
		defer docRows.Close()

		app.Documents = []Document{} // Reset documents

		for docRows.Next() {
			var doc Document
			if err := docRows.Scan(&doc.ID, &doc.Name); err != nil {
				log.Printf("Error scanning document: %v", err)
				continue
			}
			app.Documents = append(app.Documents, doc)
		}

		applications = append(applications, app)
	}

	c.JSON(http.StatusOK, applications)
})

	

// Add a new application
r.POST("/applications", func(c *gin.Context) {
	var app Application
	userID := c.DefaultQuery("user_id", "0") // Get user_id from query param (if needed)
	if err := c.ShouldBindJSON(&app); err != nil || app.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input or missing 'name'"})
		return
	}

	_, err := db.Exec("INSERT INTO applications (name, user_id) VALUES (?, ?)", app.Name, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add application"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Application added"})
})


	// Add a new document to an application
	r.POST("/documents", func(c *gin.Context) {
		var doc Document
		applicationID := c.Query("application_id")
		if err := c.ShouldBindJSON(&doc); err != nil || applicationID == "" || doc.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input or missing 'application_id' or 'name'"})
			return
		}

		_, err := db.Exec("INSERT INTO documents (application_id, name) VALUES (?, ?)", applicationID, doc.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add document"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "Document added"})
	})

// Remove an application
r.DELETE("/applications/:id", func(c *gin.Context) {
	applicationID := c.Param("id")
	userID := c.DefaultQuery("user_id", "0") // Get user_id from query param (if needed)
	
	// Verify if the application belongs to the user
	var appID int
	err := db.QueryRow("SELECT id FROM applications WHERE id = ? AND user_id = ?", applicationID, userID).Scan(&appID)
	if err != nil || appID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to delete this application"})
		return
	}
	
	_, err = db.Exec("DELETE FROM applications WHERE id = ?", applicationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove application"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Application removed"})
})


	// Remove a document
	r.DELETE("/documents/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := db.Exec("DELETE FROM documents WHERE id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove document"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Document removed"})
	})
	type User struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required"`
		Photo string `json:"photo" binding:"required"`
	}
	// Backend handling store-user (Store user in the database)
r.POST("/store-user", func(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request data"})
		return
	}

	// Insert or update user in the database
	query := `
		INSERT INTO users (name, email, photo)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
		name = VALUES(name), photo = VALUES(photo)`
	
	_, err := db.Exec(query, user.Name, user.Email, user.Photo)
	if err != nil {
		log.Println("Database error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to store user data"})
		return
	}

	// Fetch the user ID after inserting or updating the user
	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE email = ?", user.Email).Scan(&userID)
	if err != nil {
		log.Println("Database error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to retrieve user ID"})
		return
	}

	// Send back the user_id
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "User data stored successfully", "user_id": userID})
})

	r.Run(":8080") // Run the server
}
