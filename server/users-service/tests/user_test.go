package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type User struct {
	ID       string `json:"id" bson:"_id,omitempty"`
	Username string `json:"username" bson:"username"`
	Email    string `json:"email" bson:"email"`
	Password string `json:"password,omitempty" bson:"password"`
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

func TestUserRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	users := make(map[string]User)
	
	router.POST("/register", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}
		
		// Validate user data
		if user.Username == "" || user.Email == "" || user.Password == "" {
			c.JSON(400, gin.H{"error": "Missing required fields"})
			return
		}
		
		// Check if user already exists
		for _, existingUser := range users {
			if existingUser.Username == user.Username || existingUser.Email == user.Email {
				c.JSON(409, gin.H{"error": "User already exists"})
				return
			}
		}
		
		// Create user
		user.ID = fmt.Sprintf("user_%d", len(users)+1)
		user.Password = "" // Don't return password
		users[user.ID] = user
		
		c.JSON(201, gin.H{
			"message": "User created successfully",
			"user":    user,
		})
	})

	tests := []struct {
		name           string
		requestBody    User
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid user registration",
			requestBody: User{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "securepassword",
			},
			expectedStatus: 201,
		},
		{
			name: "Missing username",
			requestBody: User{
				Email:    "test@example.com",
				Password: "securepassword",
			},
			expectedStatus: 400,
			expectedError:  "Missing required fields",
		},
		{
			name: "Missing email",
			requestBody: User{
				Username: "testuser",
				Password: "securepassword",
			},
			expectedStatus: 400,
			expectedError:  "Missing required fields",
		},
		{
			name: "Missing password",
			requestBody: User{
				Username: "testuser",
				Email:    "test@example.com",
			},
			expectedStatus: 400,
			expectedError:  "Missing required fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			if tt.expectedError != "" {
				assert.Equal(t, tt.expectedError, response["error"])
			} else {
				assert.Contains(t, response, "user")
				assert.Contains(t, response, "message")
			}
		})
	}
}

func TestUserAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	// Mock user database
	users := map[string]User{
		"testuser": {
			ID:       "user_1",
			Username: "testuser",
			Email:    "test@example.com",
			Password: "hashedpassword", // In real implementation, this would be bcrypt hashed
		},
	}
	
	router.POST("/auth", func(c *gin.Context) {
		var authReq AuthRequest
		if err := c.ShouldBindJSON(&authReq); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}
		
		// Find user
		user, exists := users[authReq.Username]
		if !exists || user.Password != authReq.Password {
			c.JSON(401, gin.H{"error": "Invalid credentials"})
			return
		}
		
		// Generate JWT token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  user.ID,
			"username": user.Username,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})
		
		tokenString, err := token.SignedString([]byte("secret"))
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to generate token"})
			return
		}
		
		user.Password = "" // Don't return password
		c.JSON(200, AuthResponse{
			Token: tokenString,
			User:  user,
		})
	})

	tests := []struct {
		name           string
		requestBody    AuthRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid authentication",
			requestBody: AuthRequest{
				Username: "testuser",
				Password: "hashedpassword",
			},
			expectedStatus: 200,
		},
		{
			name: "Invalid username",
			requestBody: AuthRequest{
				Username: "nonexistent",
				Password: "hashedpassword",
			},
			expectedStatus: 401,
			expectedError:  "Invalid credentials",
		},
		{
			name: "Invalid password",
			requestBody: AuthRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			expectedStatus: 401,
			expectedError:  "Invalid credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/auth", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			if tt.expectedError != "" {
				assert.Equal(t, tt.expectedError, response["error"])
			} else {
				assert.Contains(t, response, "token")
				assert.Contains(t, response, "user")
				
				// Validate JWT token
				tokenStr := response["token"].(string)
				assert.NotEmpty(t, tokenStr)
				
				token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
					return []byte("secret"), nil
				})
				assert.NoError(t, err)
				assert.True(t, token.Valid)
			}
		})
	}
}

func TestJWTTokenValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	// Middleware for JWT validation
	router.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Missing authorization header"})
			c.Abort()
			return
		}
		
		tokenString := authHeader[7:] // Remove "Bearer " prefix
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		})
		
		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		
		claims := token.Claims.(jwt.MapClaims)
		c.Set("user_id", claims["user_id"])
		c.Next()
	})
	
	router.GET("/profile", func(c *gin.Context) {
		userID := c.GetString("user_id")
		c.JSON(200, gin.H{"user_id": userID})
	})

	// Generate valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  "user_1",
		"username": "testuser",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	validToken, _ := token.SignedString([]byte("secret"))
	
	// Generate expired token
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  "user_1",
		"username": "testuser",
		"exp":      time.Now().Add(-time.Hour).Unix(),
	})
	expiredTokenString, _ := expiredToken.SignedString([]byte("secret"))

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: 200,
		},
		{
			name:           "Missing authorization header",
			authHeader:     "",
			expectedStatus: 401,
			expectedError:  "Missing authorization header",
		},
		{
			name:           "Invalid token format",
			authHeader:     "Bearer invalid_token",
			expectedStatus: 401,
			expectedError:  "Invalid token",
		},
		{
			name:           "Expired token",
			authHeader:     "Bearer " + expiredTokenString,
			expectedStatus: 401,
			expectedError:  "Invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/profile", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			if tt.expectedError != "" {
				assert.Equal(t, tt.expectedError, response["error"])
			} else {
				assert.Contains(t, response, "user_id")
			}
		})
	}
}

func TestUserCRUDOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	users := make(map[string]User)
	
	// Create user
	router.POST("/users", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}
		
		user.ID = fmt.Sprintf("user_%d", len(users)+1)
		users[user.ID] = user
		user.Password = "" // Don't return password
		
		c.JSON(201, user)
	})
	
	// Get user
	router.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		user, exists := users[id]
		if !exists {
			c.JSON(404, gin.H{"error": "User not found"})
			return
		}
		
		user.Password = "" // Don't return password
		c.JSON(200, user)
	})
	
	// Update user
	router.PUT("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		existingUser, exists := users[id]
		if !exists {
			c.JSON(404, gin.H{"error": "User not found"})
			return
		}
		
		var updateData User
		if err := c.ShouldBindJSON(&updateData); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}
		
		// Update fields
		if updateData.Username != "" {
			existingUser.Username = updateData.Username
		}
		if updateData.Email != "" {
			existingUser.Email = updateData.Email
		}
		
		users[id] = existingUser
		existingUser.Password = "" // Don't return password
		
		c.JSON(200, existingUser)
	})
	
	// Delete user
	router.DELETE("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, exists := users[id]
		if !exists {
			c.JSON(404, gin.H{"error": "User not found"})
			return
		}
		
		delete(users, id)
		c.JSON(204, nil)
	})

	// Test Create
	t.Run("Create User", func(t *testing.T) {
		user := User{
			Username: "newuser",
			Email:    "new@example.com",
			Password: "password123",
		}
		
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		assert.Equal(t, 201, w.Code)
		
		var response User
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotEmpty(t, response.ID)
		assert.Equal(t, user.Username, response.Username)
		assert.Equal(t, user.Email, response.Email)
		assert.Empty(t, response.Password)
	})
	
	// Test Read
	t.Run("Get User", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/user_1", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		assert.Equal(t, 200, w.Code)
		
		var response User
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "user_1", response.ID)
	})
	
	// Test Update
	t.Run("Update User", func(t *testing.T) {
		updateData := User{
			Username: "updateduser",
		}
		
		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/users/user_1", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		assert.Equal(t, 200, w.Code)
		
		var response User
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "updateduser", response.Username)
	})
	
	// Test Delete
	t.Run("Delete User", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/users/user_1", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		assert.Equal(t, 204, w.Code)
		
		// Verify user is deleted
		req = httptest.NewRequest("GET", "/users/user_1", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, 404, w.Code)
	})
}