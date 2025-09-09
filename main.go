package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Cookies101-cookies/Lost-and-Found-Tracker/internal/db"
)

type Item struct {
	ID        int `gorm:"primaryKey;autoIncrement"`
	Title     string `gorm:"index"`
	Desc      string `gorm:"type:text"`
	Contact   string 
	Status    string `gorm:"index"` // "lost" or "found"
	CreatedAt time.Time `gorm:"index"`
}

var gdb *gorm.DB

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*.tmpl") // load all templates
	r.Static("/static", "./web/static")    // serve CSS (style.css)
	
	// Open DB and migrate
	database, err := db.Open("loundfound.db")
	if err != nil {
		log.Fatal("open db:", err)
	}
	if err := database.AutoMigrate(&Item{}); err != nil {
		log.Fatal("migrate:", err)
	}
	gdb = database

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

	var items []Item
	tx := gdb.Model(&Item{})

	if status == "lost" || status == "found" {
		tx = tx.Where("LOWER(status) = ?", status)
	}
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		tx = tx.Where("LOWER(title) LIKE ? OR LOWER(`desc`) LIKE ?", like, like)
	}

	if err := tx.Order("created_at DESC").Find(&items).Error; err != nil {
		c.String(http.StatusInternalServerError, "query error: %v", err)
		return
	}

	c.HTML(http.StatusOK, "index", gin.H{
		"Items": items,
		"Q": q,
		"Status": status,
		"Total": len(items),
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
		Title:     title,
		Desc:      desc,
		Contact:   contact,
		Status:    status,
		CreatedAt: time.Now(),
	}
	
	if err := gdb.Create(&item).Error; err != nil {
		c.String(http.StatusInternalServerError, "create error: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
}

func showItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid id")
		return
	}

	var it Item
	if err := gdb.First(&it, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.String(http.StatusNotFound, "item not found")
			return
		}
		c.String(http.StatusInternalServerError, "lookup error: %v", err)
		return
	}
	c.HTML(http.StatusOK, "show", gin.H{"Item": it})
}
