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
)

// BenchmarkWebSocketConnections benchmarks WebSocket connection establishment
func BenchmarkWebSocketConnections(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	
	router.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		
		// Echo messages
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			conn.WriteMessage(messageType, message)
		}
	})

	server := httptest.NewServer(router)
	defer server.Close()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		wsURL := "ws" + server.URL[4:] + "/ws"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			b.Fatalf("Failed to connect: %v", err)
		}
		conn.Close()
	}
}

// BenchmarkConcurrentWebSocketConnections benchmarks concurrent WebSocket connections
func BenchmarkConcurrentWebSocketConnections(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	
	connectionCount := int64(0)
	
	router.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		
		connectionCount++
		
		// Keep connection alive briefly
		time.Sleep(100 * time.Millisecond)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		concurrency := 10
		
		for j := 0; j < concurrency; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				wsURL := "ws" + server.URL[4:] + "/ws"
				conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
				if err != nil {
					b.Errorf("Failed to connect: %v", err)
					return
				}
				defer conn.Close()
				
				// Send a test message
				conn.WriteMessage(websocket.TextMessage, []byte("test"))
				_, _, err = conn.ReadMessage()
				if err != nil {
					b.Errorf("Failed to read message: %v", err)
				}
			}()
		}
		
		wg.Wait()
	}
	
	b.Logf("Total connections established: %d", connectionCount)
}

// BenchmarkMessageThroughput benchmarks message throughput
func BenchmarkMessageThroughput(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	
	messagesReceived := int64(0)
	
	router.GET("/ws", func(c *gin.Context) {
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
			messagesReceived++
			conn.WriteMessage(messageType, message)
		}
	})

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	message := []byte("benchmark message for throughput testing")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			b.Fatalf("Failed to send message: %v", err)
		}
		
		_, _, err = conn.ReadMessage()
		if err != nil {
			b.Fatalf("Failed to read message: %v", err)
		}
	}
	
	b.StopTimer()
	b.Logf("Messages processed: %d", messagesReceived)
	b.Logf("Messages per second: %.2f", float64(b.N)/b.Elapsed().Seconds())
}

// BenchmarkSessionCreation benchmarks session creation performance
func BenchmarkSessionCreation(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	sessionCount := 0
	
	router.POST("/session", func(c *gin.Context) {
		sessionCount++
		c.JSON(201, gin.H{
			"session_url": fmt.Sprintf("session_%d", sessionCount),
			"host":       "benchmark_user",
			"title":      "Benchmark Session",
		})
	})

	server := httptest.NewServer(router)
	defer server.Close()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		resp, err := client.Post(server.URL+"/session", "application/json", nil)
		if err != nil {
			b.Fatalf("Failed to create session: %v", err)
		}
		resp.Body.Close()
	}
	
	b.Logf("Sessions created: %d", sessionCount)
	b.Logf("Sessions per second: %.2f", float64(b.N)/b.Elapsed().Seconds())
}

// BenchmarkConcurrentSessionCreation benchmarks concurrent session creation
func BenchmarkConcurrentSessionCreation(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	var sessionCountMutex sync.Mutex
	sessionCount := 0
	
	router.POST("/session", func(c *gin.Context) {
		sessionCountMutex.Lock()
		sessionCount++
		currentCount := sessionCount
		sessionCountMutex.Unlock()
		
		// Simulate some processing time
		time.Sleep(1 * time.Millisecond)
		
		c.JSON(201, gin.H{
			"session_url": fmt.Sprintf("session_%d", currentCount),
			"host":       "benchmark_user",
			"title":      "Concurrent Benchmark Session",
		})
	})

	server := httptest.NewServer(router)
	defer server.Close()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		concurrency := 20
		
		for j := 0; j < concurrency; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Post(server.URL+"/session", "application/json", nil)
				if err != nil {
					b.Errorf("Failed to create session: %v", err)
					return
				}
				resp.Body.Close()
			}()
		}
		
		wg.Wait()
	}
	
	b.Logf("Total sessions created: %d", sessionCount)
}

// BenchmarkMemoryUsage benchmarks memory usage under load
func BenchmarkMemoryUsage(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	connections := make(map[*websocket.Conn]bool)
	var connectionsMutex sync.RWMutex
	
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	
	router.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		
		connectionsMutex.Lock()
		connections[conn] = true
		connectionsMutex.Unlock()
		
		defer func() {
			connectionsMutex.Lock()
			delete(connections, conn)
			connectionsMutex.Unlock()
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

	var connections_slice []*websocket.Conn
	
	b.ResetTimer()
	
	// Create many connections for memory testing
	for i := 0; i < b.N && i < 1000; i++ { // Limit to prevent resource exhaustion
		wsURL := "ws" + server.URL[4:] + "/ws"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			b.Fatalf("Failed to connect: %v", err)
		}
		connections_slice = append(connections_slice, conn)
	}
	
	// Hold connections briefly
	time.Sleep(100 * time.Millisecond)
	
	// Close all connections
	for _, conn := range connections_slice {
		conn.Close()
	}
	
	connectionsMutex.RLock()
	activeConnections := len(connections)
	connectionsMutex.RUnlock()
	
	b.Logf("Peak concurrent connections: %d", len(connections_slice))
	b.Logf("Remaining active connections: %d", activeConnections)
}

// BenchmarkSignallingLatency benchmarks signalling latency
func BenchmarkSignallingLatency(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	
	router.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		
		// Immediate echo for latency testing
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			// Add timestamp to measure latency
			conn.WriteMessage(messageType, append(message, []byte(fmt.Sprintf("_%d", time.Now().UnixNano()))...))
		}
	})

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	var totalLatency time.Duration
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		start := time.Now()
		message := []byte(fmt.Sprintf("latency_test_%d", i))
		
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			b.Fatalf("Failed to send message: %v", err)
		}
		
		_, _, err = conn.ReadMessage()
		if err != nil {
			b.Fatalf("Failed to read message: %v", err)
		}
		
		latency := time.Since(start)
		totalLatency += latency
	}
	
	averageLatency := totalLatency / time.Duration(b.N)
	b.Logf("Average latency: %v", averageLatency)
	b.Logf("Total messages: %d", b.N)
}