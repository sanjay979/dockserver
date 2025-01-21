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

	// Fetch all applications with documents
	r.GET("/applications", func(c *gin.Context) {
		var applications []Application
		rows, err := db.Query("SELECT id, name FROM applications")
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
	
			// Reset the documents slice for each application
			app.Documents = []Document{} // Ensure it's empty at the start
	
			for docRows.Next() {
				var doc Document
				if err := docRows.Scan(&doc.ID, &doc.Name); err != nil {
					log.Printf("Error scanning document: %v", err)
					continue
				}
				app.Documents = append(app.Documents, doc)
			}
	
			// Add the application, even if no documents are found
			applications = append(applications, app)
		}
	
		// Return the applications with an empty document array if no documents are found
		c.JSON(http.StatusOK, applications)
	})
	

	// Add a new application
	r.POST("/applications", func(c *gin.Context) {
		var app Application
		if err := c.ShouldBindJSON(&app); err != nil || app.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input or missing 'name'"})
			return
		}

		_, err := db.Exec("INSERT INTO applications (name) VALUES (?)", app.Name)
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
		id := c.Param("id")
		_, err := db.Exec("DELETE FROM applications WHERE id = ?", id)
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

	r.Run(":8080") // Run the server
}
