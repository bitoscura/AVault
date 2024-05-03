package main

import (
	"net/http"
	"os/exec"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Serve static files
	router.Static("/assets", "./assets")
	router.Static("/js", "./js")

	// Load HTML templates
	router.LoadHTMLGlob("templates/*")

	// Routes
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "form.html", nil)
	})

	router.POST("/encrypt", handleEncryption)

	// API endpoint for encryption
	router.POST("/api/encrypt", handleAPIEncryption)

	router.GET("/clear", func(c *gin.Context) {
		c.HTML(http.StatusOK, "form.html", nil)
	})

	// Start server
	router.Run(":8080")
}

func handleEncryption(c *gin.Context) {
	textToEncrypt := c.PostForm("text")
	encryptedText, err := encryptText(textToEncrypt)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error encrypting text: "+err.Error())
		return
	}
	c.HTML(http.StatusOK, "encrypted_text.html", gin.H{"EncryptedText": encryptedText})
}

func handleAPIEncryption(c *gin.Context) {
	var requestData struct {
		Text string `json:"text"`
	}

	if err := c.BindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	encryptedText, err := encryptText(requestData.Text)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error encrypting text"})
		return
	}

	c.String(http.StatusOK, encryptedText)
}

func encryptText(text string) (string, error) {
	cmd := exec.Command("ansible-vault", "encrypt_string", text)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
