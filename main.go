package main

import (
	"log"
<<<<<<< HEAD
=======
	"net/http"
	"strconv"
>>>>>>> 27f3820bde77fac602c5699e87d4d3a7d47127a5
	"time"

	"github.com/gin-gonic/gin"
)

type Item struct {
<<<<<<< HEAD
	ID        int
	Title     string
	Desc      string
	Contact   string
	Status    string // lost or found
=======
	ID int
	Title string
	Desc string
	Contact string
	Status string // "lost" or "found"
>>>>>>> 27f3820bde77fac602c5699e87d4d3a7d47127a5
	CreatedAt time.Time
}

// In-memory storage
var items = []Item{}
var nextID = 1

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*.tmpl") // for HTML we create later
<<<<<<< HEAD
	r.Static("/static", ".web/static")     // for CSS we create later
	r.Static("/uploads", "./uploads")      // placeholder for future functionality
=======
	r.Static("/static", "./web/static") // for CSS we create later
	r.Static("/uploads", "./uploads") // placeholder for future functionality
>>>>>>> 27f3820bde77fac602c5699e87d4d3a7d47127a5

	// Routes
	r.GET("/", listItems)
	r.GET("/items/new", newItemForm)
	r.POST("/items", createItem)
	r.GET("/items/:id", showItem)

<<<<<<< HEAD
	log.Println("Listening on http://localhost:8080")
	if error := r.Run(":8080"); error != nil {
		log.Fatal(error)
=======
	log.Println("listening on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
>>>>>>> 27f3820bde77fac602c5699e87d4d3a7d47127a5
	}
}

func listItems(c *gin.Context) {
<<<<<<< HEAD

}

func newItemForm(c *gin.Context) {

}

func createItem(c *gin.Context) {

}

func showItem(c *gin.Context) {

=======
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
>>>>>>> 27f3820bde77fac602c5699e87d4d3a7d47127a5
}
