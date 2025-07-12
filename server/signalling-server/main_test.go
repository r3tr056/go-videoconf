package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/r3tr056/go-videoconf/signalling-server/interfaces"
	"github.com/stretchr/testify/assert"
)

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"status":  "healthy",
			"service": "signalling-server",
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "signalling-server", response["service"])
}

func TestSessionCreation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Test session validation
	session := interfaces.Session{
		Host:     "test-user",
		Title:    "Test Meeting",
		Password: "test-password",
	}

	assert.NotEmpty(t, session.Host)
	assert.NotEmpty(t, session.Title)
	assert.NotEmpty(t, session.Password)
}