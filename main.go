package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Cookies101-cookies/Lost-and-Found-Tracker/internal/db"
)

type User struct {
	ID        int       `gorm:"primaryKey;autoIncrement"` // Unique ID
	Username  string    `gorm:"uniqueIndex;not null"`     // Username
	Email     string    `gorm:"uniqueIndex;not null"`     // Email
	Password  string    `gorm:"not null"`                 // hashed password
	CreatedAt time.Time // When the user was created
}

type Item struct {
	ID        int        `gorm:"primaryKey;autoIncrement"` // Unique ID
	Title     string     `gorm:"index"`                    // Title of the item
	Desc      string     `gorm:"type:text"`                // Description of the item
	Contact   string     // Contact info of the person who posted
	Status    string     `gorm:"index"` // "lost" or "found"
	Image     string     // URL or path to image
	CreatedAt time.Time  `gorm:"index"`             // When the post was created
	FoundAt   *time.Time `gorm:"index"`             // When the item was marked as found
	UserID    int        `gorm:"index"`             // FK to User
	User      User       `gorm:"foreignKey:UserID"` // The user who posted the item
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
	if err := database.AutoMigrate(&User{}, &Item{}); err != nil {
		log.Fatal("migrate:", err)
	}
	gdb = database

	// Attach user middleware
	r.Use(attachUser())

	// Routes
	r.GET("/", listItems)
	r.GET("/items/new", newItemForm)
	r.POST("/items", createItem)
	r.POST("/items/:id/mark-found", markAsFound)
	r.GET("/register", showRegisterForm)
	r.POST("/register", registerUser)
	r.GET("/login", showLoginForm)
	r.POST("/login", loginUser)
	r.GET("/logout", logoutUser)

	// Edit and show routes
	r.GET("/items/:id/edit", editItemForm)
	r.POST("/items/:id/edit", updateItem)
	r.GET("/items/:id", showItem)

	// Route to delete all items from database only
	r.GET("/clear-db", func(c *gin.Context) {
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

	c.HTML(http.StatusOK, "index", getTemplateData(c, gin.H{
		"Items":  items,
		"Q":      q,
		"Status": status,
		"Total":  len(items),
		"Title":  "All Items",
	}))
}

// Show form for creating new item
func newItemForm(c *gin.Context) {
	c.HTML(http.StatusOK, "new", getTemplateData(c, gin.H{"Title": "Post New Item"}))
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

	user, err := getCurrentUser(c)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	item := Item{
		Title:     title,
		Desc:      desc,
		Contact:   contact,
		Status:    status,
		Image:     filename,
		CreatedAt: time.Now(),
		UserID:    user.ID,
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
	c.HTML(http.StatusOK, "show", getTemplateData(c, gin.H{"Item": it, "Title": it.Title}))
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

	user, err := getCurrentUser(c)
	if err != nil || it.UserID != user.ID {
		c.String(http.StatusForbidden, "You cannot edit this item")
		return
	}

	c.HTML(http.StatusOK, "edit", getTemplateData(c, gin.H{"Item": it, "Title": "Edit Item"}))
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

	user, err := getCurrentUser(c)
	if err != nil || it.UserID != user.ID {
		c.String(http.StatusForbidden, "You cannot edit this item")
		return
	}

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

	now := time.Now()
	it.Status = "found"
	it.FoundAt = &now

	if err := gdb.Save(&it).Error; err != nil {
		c.String(http.StatusInternalServerError, "failed to mark item as found: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/items/"+idStr)
}

// Get current logged-in user
func getCurrentUser(c *gin.Context) (*User, error) {
	cookie, err := c.Cookie("user_id")
	if err != nil {
		return nil, err
	}
	uid, _ := strconv.Atoi(cookie)
	var user User
	if err := gdb.First(&user, uid).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Middleware to attach user to context
func attachUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getCurrentUser(c)
		if err == nil {
			c.Set("CurrentUser", user)
		}
		c.Next()
	}
}

func getTemplateData(c *gin.Context, extra gin.H) gin.H {
	data := gin.H{}

	for k, v := range extra {
		data[k] = v
	}

	if user, exists := c.Get("CurrentUser"); exists {
		data["CurrentUser"] = user
	} else {
		data["CurrentUser"] = nil
	}
	return data
}

// Has password using bcrypt
func hashPassword(pw string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(bytes), err
}

// Check password using bcrypt
func checkPassword(hash, pw string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
	return err == nil
}

// Show login form
func showRegisterForm(c *gin.Context) {
	c.HTML(http.StatusOK, "register", getTemplateData(c, gin.H{"Title": "Register"}))
}

// Handle user registration
func registerUser(c *gin.Context) {
	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	confirm := c.PostForm("confirm")

	if username == "" || email == "" || password == "" || password != confirm {
		c.String(http.StatusBadRequest, "Invalid input or passwords do not match")
		return
	}

	hashed, _ := hashPassword(password)

	user := User{
		Username:  username,
		Email:     email,
		Password:  hashed,
		CreatedAt: time.Now(),
	}
	if err := gdb.Create(&user).Error; err != nil {
		c.String(http.StatusInternalServerError, "Could not create user: %v", err)
		return
	}

	// Set cookie
	c.SetCookie("user_id", strconv.Itoa(user.ID), 3600*24, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/")
}

// Show login form
func showLoginForm(c *gin.Context) {
	c.HTML(http.StatusOK, "login", getTemplateData(c, gin.H{"Title": "Login"}))
}

// Handle user login
func loginUser(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	var user User
	if err := gdb.Where("email = ?", email).First(&user).Error; err != nil {
		c.String(http.StatusUnauthorized, "Invalid email or password")
		return
	}

	if !checkPassword(user.Password, password) {
		c.String(http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Set cookie
	c.SetCookie("user_id", strconv.Itoa(user.ID), 3600*24, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/")
}

// Handle user logout
func logoutUser(c *gin.Context) {
	// Clear cookie
	c.SetCookie("user_id", "", -1, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/")
}
