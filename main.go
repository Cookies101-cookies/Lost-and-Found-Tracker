package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Item struct {
	ID int
	Title string
	Desc string
	Contact string
	Status string // "lost" or "found"
	CreatedAt time.Time
}

// In-memory storage
var items = []Item{}
var nextID = 1

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*.tmpl") // for HTML we create later
	r.Static("/static", "./web/static") // for CSS we create later
	r.Static("/uploads", "./uploads") // placeholder for future functionality

	// Routes
	r.GET("/", listItems)
	r.GET("/items/new", newItemForm)
	r.POST("/items", createItem)
	r.GET("/items/:id", showItem)

	log.Println("listening on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func listItems(c *gin.Context) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"Items": items,
	})
}

func newItemForm(c *gin.Context) {
	c.HTML(http.StatusOK, "new.tmpl", nil)
}

func createItem(c *gin.Context) {
	title := c.PostForm("title")
	desc := c.PostForm("desc")
	contact := c.PostForm("contact")
	status := c.PostForm("status")

	if status != "lost" && status != "found" {
		status = "lost"
	}

	if title == "" || contact == "" {
		c.String(http.StatusBadRequest, "title and contact are required")
		return
	}

	item := Item{
		ID: nextID,
		Title: title,
		Desc: desc,
		Contact: contact,
		Status: status,
		CreatedAt: time.Now(),
	}
	nextID++;
	items = append(items, item)

	c.Redirect(http.StatusSeeOther, "/")
}

func showItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid id")
		return
	}
	for _, it := range items {
		if it.ID == id {
			c.HTML(http.StatusOK, "show.tmpl", gin.H{"Item": it})
			return
		}
	}
	c.String(http.StatusNotFound, "item not found")
}
