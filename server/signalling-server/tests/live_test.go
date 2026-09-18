package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/r3tr056/go-videoconf/signalling-server/controllers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func liveEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("VIDEOCONF_LIVE_MONGO") == "" {
		t.Skip("live mongo test: set VIDEOCONF_LIVE_MONGO=1 with mongo on DB_HOST:DB_PORT")
	}
}

// liveClient dials mongo with the same variables the service uses.
func liveClient(t *testing.T) *mongo.Client {
	t.Helper()
	host, port, user, pass :=
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USERNAME"), os.Getenv("DB_PASSWORD")
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "27017"
	}
	if user == "" {
		user = "root"
	}
	if pass == "" {
		pass = "rootpassword"
	}
	credential := options.Credential{Username: user, Password: pass, AuthSource: "admin"}
	clientOptions := options.Client().ApplyURI("mongodb://" + host + ":" + port).SetAuth(credential)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	require.NoError(t, err)
	require.NoError(t, client.Ping(ctx, readpref.Primary()))
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	return client
}

// liveRouter wires the real controllers exactly like main does.
func liveRouter(client *mongo.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("db", client)
		c.Next()
	})
	router.POST("/session", controllers.CreateSession)
	router.GET("/connect", controllers.GetSession)
	router.POST("/connect/:url", controllers.ConnectSession)
	return router
}

func liveDrop(t *testing.T, client *mongo.Client) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, client.Database("vidchat").Drop(ctx))
}

func doJSON(t *testing.T, router *gin.Engine, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var decoded map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &decoded)
	return w.Code, decoded
}

func TestLiveSessionJoinFlow(t *testing.T) {
	liveEnv(t)
	client := liveClient(t)
	controllers.EnsureIndexes(client)
	defer liveDrop(t, client)
	router := liveRouter(client)

	code, created := doJSON(t, router, "POST", "/session", map[string]string{
		"host": "live_host", "title": "Live Meeting", "password": "secret",
	})
	require.Equal(t, http.StatusOK, code)
	socketURL, _ := created["socket"].(string)
	require.NotEmpty(t, socketURL)

	code, joined := doJSON(t, router, "POST", "/connect/"+socketURL, map[string]string{
		"password": "secret",
	})
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "Live Meeting", joined["title"])
	assert.NotEmpty(t, joined["socket"])

	code, _ = doJSON(t, router, "POST", "/connect/"+socketURL, map[string]string{
		"password": "wrong",
	})
	assert.Equal(t, http.StatusBadRequest, code)

	code, _ = doJSON(t, router, "POST", "/connect/does-not-exist", map[string]string{
		"password": "secret",
	})
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestLiveConcurrentSessionStorm(t *testing.T) {
	liveEnv(t)
	client := liveClient(t)
	controllers.EnsureIndexes(client)
	defer liveDrop(t, client)
	router := liveRouter(client)

	const workers, iterations = 20, 25
	var wg sync.WaitGroup
	errs := make(chan error, workers*iterations)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				title := fmt.Sprintf("storm-%d-%d", w, i)
				code, created := doJSON(t, router, "POST", "/session", map[string]string{
					"host": fmt.Sprintf("host-%d", w), "title": title, "password": "pw",
				})
				if code != http.StatusOK {
					errs <- fmt.Errorf("worker %d create %s: status %d", w, title, code)
					return
				}
				socketURL, _ := created["socket"].(string)
				code, _ = doJSON(t, router, "POST", "/connect/"+socketURL, map[string]string{
					"password": "pw",
				})
				if code != http.StatusOK {
					errs <- fmt.Errorf("worker %d join %s: status %d", w, title, code)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

func TestLiveSocketIndexUnique(t *testing.T) {
	liveEnv(t)
	client := liveClient(t)
	controllers.EnsureIndexes(client)
	defer liveDrop(t, client)
	router := liveRouter(client)

	body := map[string]string{"host": "dup_host", "title": "Dup Meeting", "password": "pw"}
	code, _ := doJSON(t, router, "POST", "/session", body)
	require.Equal(t, http.StatusOK, code)

	// Same host+title hashes to the same public URL: the unique index rejects it.
	code, _ = doJSON(t, router, "POST", "/session", body)
	assert.Equal(t, http.StatusInternalServerError, code)
}

func TestLiveSocketIndexExists(t *testing.T) {
	liveEnv(t)
	client := liveClient(t)
	controllers.EnsureIndexes(client)
	defer liveDrop(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cursor, err := client.Database("vidchat").Collection("sockets").Indexes().List(ctx)
	require.NoError(t, err)
	defer cursor.Close(ctx)

	names := map[string]bool{}
	for cursor.Next(ctx) {
		var spec bson.M
		require.NoError(t, cursor.Decode(&spec))
		if name, ok := spec["name"].(string); ok {
			names[name] = true
		}
	}
	assert.True(t, names["hashedUrl_1"], "expected hashedUrl_1 index, got %v", names)
}
