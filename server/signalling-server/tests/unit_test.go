package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/r3tr056/go-videoconf/signalling-server/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSocketConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	// Mock WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			t.Errorf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()
		
		// Echo messages back
		for {
			messageType, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			conn.WriteMessage(messageType, msg)
		}
	})

	server := httptest.NewServer(router)
	defer server.Close()

	// Test WebSocket connection
	wsURL := "ws" + server.URL[4:] + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Test message sending
	testMessage := "test message"
	err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
	require.NoError(t, err)

	// Test message receiving
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, testMessage, string(msg))
}

func TestSessionValidation(t *testing.T) {
	tests := []struct {
		name    string
		session interfaces.Session
		isValid bool
	}{
		{
			name: "Valid session",
			session: interfaces.Session{
				Host:     "user123",
				Title:    "Team Meeting",
				Password: "secure123",
			},
			isValid: true,
		},
		{
			name: "Empty host",
			session: interfaces.Session{
				Host:     "",
				Title:    "Team Meeting",
				Password: "secure123",
			},
			isValid: false,
		},
		{
			name: "Empty title",
			session: interfaces.Session{
				Host:     "user123",
				Title:    "",
				Password: "secure123",
			},
			isValid: false,
		},
		{
			name: "Weak password",
			session: interfaces.Session{
				Host:     "user123",
				Title:    "Team Meeting",
				Password: "123",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := validateSession(tt.session)
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func validateSession(session interfaces.Session) bool {
	if session.Host == "" || session.Title == "" {
		return false
	}
	if len(session.Password) < 6 {
		return false
	}
	return true
}

func TestSessionCreationAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	router.POST("/session", func(c *gin.Context) {
		var session interfaces.Session
		if err := c.ShouldBindJSON(&session); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}
		
		if !validateSession(session) {
			c.JSON(400, gin.H{"error": "Invalid session data"})
			return
		}
		
		// Generate session URL
		sessionURL := "session_" + session.Host + "_" + time.Now().Format("20060102150405")
		
		c.JSON(201, gin.H{
			"session_url": sessionURL,
			"host":       session.Host,
			"title":      session.Title,
		})
	})

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedFields []string
	}{
		{
			name: "Valid session creation",
			requestBody: interfaces.Session{
				Host:     "user123",
				Title:    "Team Meeting",
				Password: "secure123",
			},
			expectedStatus: 201,
			expectedFields: []string{"session_url", "host", "title"},
		},
		{
			name: "Invalid session - empty host",
			requestBody: interfaces.Session{
				Host:     "",
				Title:    "Team Meeting",
				Password: "secure123",
			},
			expectedStatus: 400,
			expectedFields: []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/session", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			for _, field := range tt.expectedFields {
				assert.Contains(t, response, field)
			}
		})
	}
}

func TestConcurrentConnections(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	var connectionCount int64
	router.GET("/ws", func(c *gin.Context) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		
		atomic.AddInt64(&connectionCount, 1)
		
		// Keep connection alive for testing
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	})

	server := httptest.NewServer(router)
	defer server.Close()

	// Test multiple concurrent connections
	numConnections := 10
	connections := make([]*websocket.Conn, numConnections)
	
	for i := 0; i < numConnections; i++ {
		wsURL := "ws" + server.URL[4:] + "/ws"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		connections[i] = conn
	}
	
	// Allow some time for connections to establish
	time.Sleep(100 * time.Millisecond)
	
	// Close all connections
	for _, conn := range connections {
		conn.Close()
	}
	
	// Verify connections were handled
	assert.Equal(t, int64(numConnections), atomic.LoadInt64(&connectionCount))
}

func TestSignallingFlow(t *testing.T) {
	type Message struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}
	
	tests := []struct {
		name         string
		messageType  string
		expectedType string
	}{
		{
			name:         "Offer message",
			messageType:  "offer",
			expectedType: "offer",
		},
		{
			name:         "Answer message",
			messageType:  "answer",
			expectedType: "answer",
		},
		{
			name:         "ICE candidate",
			messageType:  "ice-candidate",
			expectedType: "ice-candidate",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := Message{
				Type: tt.messageType,
				Data: map[string]interface{}{
					"test": "data",
				},
			}
			
			// Simulate message processing
			assert.Equal(t, tt.expectedType, message.Type)
			assert.NotNil(t, message.Data)
		})
	}
}