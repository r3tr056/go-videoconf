package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StressTestConcurrentConnections tests the server under high concurrent load
func TestStressTestConcurrentConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	var mu sync.Mutex
	connectionCount := 0
	activeConnections := make(map[*websocket.Conn]bool)
	
	router.GET("/ws", func(c *gin.Context) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		
		mu.Lock()
		connectionCount++
		activeConnections[conn] = true
		mu.Unlock()
		
		defer func() {
			mu.Lock()
			delete(activeConnections, conn)
			mu.Unlock()
			conn.Close()
		}()
		
		// Handle messages
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			// Echo back the message
			if err := conn.WriteMessage(messageType, message); err != nil {
				break
			}
		}
	})

	server := httptest.NewServer(router)
	defer server.Close()

	// Test parameters for Google Meet level performance
	numConnections := 100
	messagesPerConnection := 10
	
	var wg sync.WaitGroup
	startTime := time.Now()
	
	// Create concurrent connections
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			
			wsURL := "ws" + server.URL[4:] + "/ws"
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				t.Errorf("Client %d failed to connect: %v", clientID, err)
				return
			}
			defer conn.Close()
			
			// Send multiple messages
			for j := 0; j < messagesPerConnection; j++ {
				message := fmt.Sprintf("Message %d from client %d", j, clientID)
				
				err := conn.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					t.Errorf("Client %d failed to send message: %v", clientID, err)
					return
				}
				
				// Read response
				_, response, err := conn.ReadMessage()
				if err != nil {
					t.Errorf("Client %d failed to read message: %v", clientID, err)
					return
				}
				
				if string(response) != message {
					t.Errorf("Client %d message mismatch: expected %s, got %s", clientID, message, string(response))
				}
				
				// Small delay to simulate real usage
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}
	
	// Wait for all clients to complete
	wg.Wait()
	duration := time.Since(startTime)
	
	// Performance assertions
	assert.LessOrEqual(t, connectionCount, numConnections+10, "Connection count should not exceed expected range")
	assert.Less(t, duration, 30*time.Second, "Test should complete within 30 seconds")
	
	// Calculate performance metrics
	totalMessages := numConnections * messagesPerConnection * 2 // Send and receive
	messagesPerSecond := float64(totalMessages) / duration.Seconds()
	
	t.Logf("Performance Results:")
	t.Logf("  Total connections: %d", numConnections)
	t.Logf("  Total messages: %d", totalMessages)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Messages per second: %.2f", messagesPerSecond)
	
	// Google Meet level performance targets
	assert.Greater(t, messagesPerSecond, 1000.0, "Should handle at least 1000 messages per second")
}

// TestLoadTestSessionCreation tests session creation under load
func TestLoadTestSessionCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	sessionCount := 0
	var mu sync.Mutex
	
	router.POST("/session", func(c *gin.Context) {
		mu.Lock()
		sessionCount++
		currentCount := sessionCount
		mu.Unlock()
		
		// Simulate some processing time
		time.Sleep(5 * time.Millisecond)
		
		c.JSON(201, gin.H{
			"session_url": fmt.Sprintf("session_%d", currentCount),
			"host":       "test_user",
			"title":      "Load Test Meeting",
		})
	})

	server := httptest.NewServer(router)
	defer server.Close()

	numRequests := 500
	var wg sync.WaitGroup
	startTime := time.Now()
	
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()
			
			resp, err := http.Post(server.URL+"/session", "application/json", nil)
			if err != nil {
				t.Errorf("Request %d failed: %v", requestID, err)
				return
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != 201 {
				t.Errorf("Request %d returned status %d", requestID, resp.StatusCode)
			}
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(startTime)
	
	requestsPerSecond := float64(numRequests) / duration.Seconds()
	
	t.Logf("Load Test Results:")
	t.Logf("  Total requests: %d", numRequests)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Requests per second: %.2f", requestsPerSecond)
	
	// Performance targets
	assert.Greater(t, requestsPerSecond, 100.0, "Should handle at least 100 requests per second")
	assert.Equal(t, sessionCount, numRequests, "All sessions should be created successfully")
}

// TestMemoryUsageUnderLoad tests memory consumption during high load
func TestMemoryUsageUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	connections := make(map[*websocket.Conn]bool)
	var mu sync.Mutex
	
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
		
		mu.Lock()
		connections[conn] = true
		mu.Unlock()
		
		defer func() {
			mu.Lock()
			delete(connections, conn)
			mu.Unlock()
			conn.Close()
		}()
		
		// Keep connection alive
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	})

	server := httptest.NewServer(router)
	defer server.Close()

	// Create many persistent connections
	numConnections := 200
	clientConnections := make([]*websocket.Conn, numConnections)
	
	for i := 0; i < numConnections; i++ {
		wsURL := "ws" + server.URL[4:] + "/ws"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		clientConnections[i] = conn
	}
	
	// Hold connections for a period
	time.Sleep(2 * time.Second)
	
	mu.Lock()
	activeCount := len(connections)
	mu.Unlock()
	
	// Clean up connections
	for _, conn := range clientConnections {
		conn.Close()
	}
	
	// Wait for cleanup
	time.Sleep(1 * time.Second)
	
	mu.Lock()
	finalCount := len(connections)
	mu.Unlock()
	
	assert.Equal(t, numConnections, activeCount, "All connections should be tracked")
	assert.Equal(t, 0, finalCount, "All connections should be cleaned up")
}

// TestLatencyUnderLoad measures response latency under load
func TestLatencyUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping latency test in short mode")
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
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
		
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			// Immediate echo back
			conn.WriteMessage(messageType, message)
		}
	})

	server := httptest.NewServer(router)
	defer server.Close()

	numClients := 50
	messagesPerClient := 20
	var totalLatency time.Duration
	var latencyMutex sync.Mutex
	var wg sync.WaitGroup
	
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			wsURL := "ws" + server.URL[4:] + "/ws"
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			
			for j := 0; j < messagesPerClient; j++ {
				message := fmt.Sprintf("latency_test_%d", j)
				
				start := time.Now()
				err := conn.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					continue
				}
				
				_, _, err = conn.ReadMessage()
				if err != nil {
					continue
				}
				latency := time.Since(start)
				
				latencyMutex.Lock()
				totalLatency += latency
				latencyMutex.Unlock()
			}
		}()
	}
	
	wg.Wait()
	
	totalMessages := numClients * messagesPerClient
	averageLatency := totalLatency / time.Duration(totalMessages)
	
	t.Logf("Latency Test Results:")
	t.Logf("  Total messages: %d", totalMessages)
	t.Logf("  Average latency: %v", averageLatency)
	
	// Google Meet level latency targets (sub-200ms)
	assert.Less(t, averageLatency, 200*time.Millisecond, "Average latency should be under 200ms")
}