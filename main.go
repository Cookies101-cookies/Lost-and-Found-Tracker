package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

type Item struct {
	ID        int
	Title     string
	Desc      string
	Contact   string
	Status    string // lost or found
	CreatedAt time.Time
}

// In-memory storage
var items = []Item{}
var nextID = 1

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*.tmpl") // for HTML we create later
	r.Static("/static", ".web/static")     // for CSS we create later
	r.Static("/uploads", "./uploads")      // placeholder for future functionality

	// Routes
	r.GET("/", listItems)
	r.GET("/items/new", newItemForm)
	r.POST("/items", createItem)
	r.GET("/items/:id", showItem)

	log.Println("Listening on http://localhost:8080")
	if error := r.Run(":8080"); error != nil {
		log.Fatal(error)
	}
}

func listItems(c *gin.Context) {

}

func newItemForm(c *gin.Context) {

}

func createItem(c *gin.Context) {

}

func showItem(c *gin.Context) {

}
