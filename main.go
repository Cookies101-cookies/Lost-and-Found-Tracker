package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Item struct {
	ID        int
	Title     string
	Desc      string
	Contact   string
	Status    string // "lost" or "found"
	CreatedAt time.Time
}

// In-memory storage
var items = []Item{}
var nextID = 1
var htmlTemplate *gin.Engine

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*.tmpl") // load all templates
	r.Static("/static", "./web/static")    // serve CSS (style.css)
	htmlTemplate = r

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
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))

	filtered := make([]Item, 0, len(items))
	for _, it := range items {
		// status filter
		if status == "lost" || status == "found" {
			if strings.ToLower(it.Status) != status {
				continue
			}
		}
		// text filter
		if q != "" {
			title := strings.ToLower(it.Title)
			desc := strings.ToLower(it.Desc)
			if !strings.Contains(title, q) && !strings.Contains(desc, q) {
				continue
			}
		}
		filtered = append(filtered, it)
	}

	log.Printf("listItems q=%q status=%q total=%d", q, status, len(filtered))
	c.HTML(http.StatusOK, "index", gin.H{
		"Title":  "All Items",
		"Items":  filtered,
		"Q":      c.Query("q"), // preserve original casing for display
		"Status": status,
		"Total":  len(filtered),
	})
}

func newItemForm(c *gin.Context) {
	c.HTML(http.StatusOK, "new", gin.H{
		"Title": "Post New Item",
	})
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
		ID:        nextID,
		Title:     title,
		Desc:      desc,
		Contact:   contact,
		Status:    status,
		CreatedAt: time.Now(),
	}
	nextID++
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
			c.HTML(http.StatusOK, "show", gin.H{
				"Title": "Item Details",
				"Item":  it,
			})
			return
		}
	}
	c.String(http.StatusNotFound, "item not found")
}
