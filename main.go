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
	ID        int    `gorm:"primaryKey;autoIncrement"`
	Title     string `gorm:"index"`
	Desc      string `gorm:"type:text"`
	Contact   string
	Status    string    `gorm:"index"` // "lost" or "found"
	Image     string    // URL or path to image
	CreatedAt time.Time `gorm:"index"`
}

var gdb *gorm.DB

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*.tmpl") // load all templates
	r.Static("/static", "./web/static")    // serve CSS and uploads

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
	r.POST("/items/:id/mark-found", markAsFound)

	// Edit and show routes
	r.GET("/items/:id/edit", editItemForm)
	r.POST("/items/:id/edit", updateItem)
	r.GET("/items/:id", showItem)

	// Route to delete all items from database only
	r.GET("/clear-db-only", func(c *gin.Context) {
		// Delete all items from the database, but keep uploaded images
		if err := gdb.Where("1 = 1").Delete(&Item{}).Error; err != nil {
			c.String(http.StatusInternalServerError, "Failed to clear items: %v", err)
			return
		}

		c.String(http.StatusOK, "All database items cleared! Uploaded images remain.")
	})

	r.LoadHTMLGlob("web/templates/*.tmpl")
	log.Println("Templates loaded")

	log.Println("listening on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

// List all items
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
		"Items":  items,
		"Q":      q,
		"Status": status,
		"Total":  len(items),
	})
}

// Show form for creating new item
func newItemForm(c *gin.Context) {
	c.HTML(http.StatusOK, "new", gin.H{"Title": "Post New Item"})
}

// Create new item
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

	var filename string
	file, err := c.FormFile("image")
	if err == nil {
		filename = strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + file.Filename
		if err := c.SaveUploadedFile(file, "./web/static/uploads/"+filename); err != nil {
			c.String(http.StatusInternalServerError, "failed to save image: %v", err)
			return
		}
	}

	item := Item{
		Title:     title,
		Desc:      desc,
		Contact:   contact,
		Status:    status,
		Image:     filename,
		CreatedAt: time.Now(),
	}

	if err := gdb.Create(&item).Error; err != nil {
		c.String(http.StatusInternalServerError, "create error: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
}

// Show single item
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

// Show form for editing an item
func editItemForm(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	var it Item
	if err := gdb.First(&it, id).Error; err != nil {
		c.String(http.StatusNotFound, "item not found")
		return
	}

	c.HTML(http.StatusOK, "edit", gin.H{"Item": it})
	log.Println("Edit page rendered for ID:", id)
}

// Update item after editing
func updateItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Println("Invalid item ID:", idStr)
		c.String(http.StatusBadRequest, "invalid item ID")
		return
	}

	// Fetch the item
	var it Item
	if err := gdb.First(&it, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("Item not found:", id)
			c.String(http.StatusNotFound, "item not found")
			return
		}
		log.Println("DB lookup error:", err)
		c.String(http.StatusInternalServerError, "lookup error: %v", err)
		return
	}

	log.Println("Editing item:", it)

	// Get form values
	title := c.PostForm("title")
	desc := c.PostForm("desc")
	contact := c.PostForm("contact")
	status := c.PostForm("status")

	log.Println("Received form data - Title:", title, "Desc:", desc, "Contact:", contact, "Status:", status)

	// Validate required fields
	if title == "" || contact == "" {
		log.Println("Missing required fields")
		c.String(http.StatusBadRequest, "title and contact are required")
		return
	}

	// Update fields
	it.Title = title
	it.Desc = desc
	it.Contact = contact
	if status == "lost" || status == "found" {
		it.Status = status
	}

	// Handle optional image upload
	file, err := c.FormFile("image")
	if err == nil {
		filename := strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + file.Filename
		savePath := "./web/static/uploads/" + filename
		if saveErr := c.SaveUploadedFile(file, savePath); saveErr != nil {
			log.Println("Failed to save uploaded image:", saveErr)
			c.String(http.StatusInternalServerError, "failed to save image: %v", saveErr)
			return
		}
		it.Image = filename
		log.Println("Uploaded new image:", filename)
	} else {
		log.Println("No new image uploaded")
	}

	// Save changes to DB
	if err := gdb.Save(&it).Error; err != nil {
		log.Println("Failed to update item:", err)
		c.String(http.StatusInternalServerError, "failed to update item: %v", err)
		return
	}

	log.Println("Item updated successfully:", it)
	c.Redirect(http.StatusSeeOther, "/items/"+idStr)
}

// Mark item as found
func markAsFound(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid item ID")
		return
	}

	var it Item
	if err := gdb.First(&it, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.String(http.StatusNotFound, "item not found")
			return
		}
		c.String(http.StatusInternalServerError, "DB error: %v", err)
		return
	}

	if it.Status == "found" {
		c.String(http.StatusBadRequest, "item is already marked as found")
		return
	}

	it.Status = "found"
	if err := gdb.Save(&it).Error; err != nil {
		c.String(http.StatusInternalServerError, "failed to update item: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/items/"+idStr)
}
