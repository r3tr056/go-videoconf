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

	"github.com/gin-gonic/gin"
	"github.com/r3tr056/go-videoconf/users-service/common"
	"github.com/r3tr056/go-videoconf/users-service/controllers"
	"github.com/r3tr056/go-videoconf/users-service/dao"
	"github.com/r3tr056/go-videoconf/users-service/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// liveDB points the service at an isolated test database. It skips unless
// VIDEOCONF_LIVE_MONGO=1 with mongo reachable via DB_HOST/DB_PORT/DB_USERNAME/
// DB_PASSWORD (same variables the service uses). Callers defer cleanup.
func liveDB(t *testing.T) func() {
	t.Helper()
	if os.Getenv("VIDEOCONF_LIVE_MONGO") == "" {
		t.Skip("live mongo test: set VIDEOCONF_LIVE_MONGO=1 with mongo on DB_HOST:DB_PORT")
	}
	prevName, prevUser, prevPass, prevAddr :=
		common.MgDBName, common.MgUsername, common.MgPassword, common.MgAddress
	common.MgDBName = "vidchat_live_test"
	if u := os.Getenv("DB_USERNAME"); u != "" {
		common.MgUsername = u
	}
	if p := os.Getenv("DB_PASSWORD"); p != "" {
		common.MgPassword = p
	}
	host, port := os.Getenv("DB_HOST"), os.Getenv("DB_PORT")
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "27017"
	}
	common.MgAddress = host + ":" + port

	require.NoError(t, database.Database.Init())
	return func() {
		_ = database.Database.Client.Database(common.MgDBName).Drop(context.Background())
		database.Database.Close()
		common.MgDBName, common.MgUsername, common.MgPassword, common.MgAddress =
			prevName, prevUser, prevPass, prevAddr
	}
}

func TestLiveUserCRUDRoundTrip(t *testing.T) {
	cleanup := liveDB(t)
	defer cleanup()

	userDao := &dao.User{}

	// Seed: the admin user exists after init on an empty database.
	admin, err := userDao.FindByName("admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", admin.Name)

	// Create -> read -> delete through the DAO.
	id := primitive.NewObjectID()
	_, err = database.Database.Users().InsertOne(context.Background(), database.UserModel{
		ID: id, Name: "live_user", Password: "pw",
	})
	require.NoError(t, err)

	got, err := userDao.GetByID(id.Hex())
	require.NoError(t, err)
	assert.Equal(t, "live_user", got.Name)

	byName, err := userDao.FindByName("live_user")
	require.NoError(t, err)
	assert.Equal(t, id, byName.ID)

	require.NoError(t, userDao.DeleteByID(id.Hex()))
	_, err = userDao.GetByID(id.Hex())
	require.Error(t, err, "deleted user must not be found")
}

func TestLiveUserHTTPFlow(t *testing.T) {
	cleanup := liveDB(t)
	defer cleanup()

	prevSecret := common.JwtSecretPassword
	common.JwtSecretPassword = "live-test-secret"
	defer func() { common.JwtSecretPassword = prevSecret }()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctl := &controllers.User{}
	router.POST("/users", ctl.CreateUser)
	router.GET("/users/:id", ctl.GetUser)
	router.POST("/auth", ctl.Authenticate)
	router.PUT("/users/:id", ctl.UpdateUser)
	router.DELETE("/users/:id", ctl.DeleteUser)

	do := func(method, path string, body any) (int, map[string]any) {
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

	code, created := do("POST", "/users", map[string]string{"name": "http_user", "password": "pw"})
	require.Equal(t, http.StatusCreated, code)
	uid, _ := created["id"].(string)
	require.NotEmpty(t, uid)

	code, fetched := do("GET", "/users/"+uid, nil)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "http_user", fetched["name"])
	assert.Equal(t, uid, fetched["id"], "id keeps the hex-string shape")

	code, authed := do("POST", "/auth", map[string]string{"username": "http_user", "password": "pw"})
	require.Equal(t, http.StatusOK, code)
	assert.NotEmpty(t, authed["token"])

	code, _ = do("POST", "/auth", map[string]string{"username": "http_user", "password": "wrong"})
	assert.Equal(t, http.StatusUnauthorized, code)

	code, _ = do("PUT", "/users/"+uid, map[string]string{"name": "http_user2", "password": "pw2"})
	assert.Equal(t, http.StatusOK, code)

	code, _ = do("DELETE", "/users/"+uid, nil)
	assert.Equal(t, http.StatusOK, code)

	code, _ = do("GET", "/users/"+uid, nil)
	assert.Equal(t, http.StatusNotFound, code)
}

func TestLiveConcurrentStorm(t *testing.T) {
	cleanup := liveDB(t)
	defer cleanup()

	userDao := &dao.User{}
	const workers, iterations = 20, 25
	var wg sync.WaitGroup
	errs := make(chan error, workers*iterations)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				if _, err := userDao.FindByName("admin"); err != nil {
					errs <- fmt.Errorf("worker %d find: %w", w, err)
					return
				}
				if _, err := userDao.GetAll(); err != nil {
					errs <- fmt.Errorf("worker %d list: %w", w, err)
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

func TestLiveNameIndexExists(t *testing.T) {
	cleanup := liveDB(t)
	defer cleanup()

	cursor, err := database.Database.Users().Indexes().List(context.Background())
	require.NoError(t, err)
	defer cursor.Close(context.Background())

	names := map[string]bool{}
	for cursor.Next(context.Background()) {
		var spec bson.M
		require.NoError(t, cursor.Decode(&spec))
		if name, ok := spec["name"].(string); ok {
			names[name] = true
		}
	}
	assert.True(t, names["name_1"], "expected name_1 index, got %v", names)
}
