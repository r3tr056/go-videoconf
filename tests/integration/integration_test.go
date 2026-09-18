// Package integration exercises the live signalling + users backends over
// their real HTTP/WebSocket contracts: session create/join, the
// /ws/:socket signalling flow, and the users CRUD + auth API.
//
// It requires both services (and MongoDB) running. Bases come from
// SIGNALLING_SERVER_URL / USERS_SERVICE_URL with localhost defaults:
//
//	POST /session | GET /connect?url= | POST /connect/:url | /ws/:socket
//	POST /users | GET /users | POST /auth | PUT/DELETE /users/:id
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func signallingBase() string { return envOr("SIGNALLING_SERVER_URL", "http://localhost:8080") }
func usersBase() string      { return envOr("USERS_SERVICE_URL", "http://localhost:8081") }
func signallingWS() string {
	return strings.Replace(signallingBase(), "http", "ws", 1)
}

// requireLive skips when backends are out of scope (short mode) instead of
// failing on connection refused.
func requireLive(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping live-backend test in short mode")
	}
}

func unique(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func postJSON(t *testing.T, url string, body any) (int, map[string]any) {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(raw))
	require.NoError(t, err, "POST %s unreachable (backends running?)", url)
	defer resp.Body.Close()
	var decoded map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&decoded))
	return resp.StatusCode, decoded
}

func postRaw(t *testing.T, url, contentType string, raw string) int {
	t.Helper()
	resp, err := httpClient.Post(url, contentType, strings.NewReader(raw))
	require.NoError(t, err)
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

func getBody(t *testing.T, url string) (int, []byte) {
	t.Helper()
	resp, err := httpClient.Get(url)
	require.NoError(t, err, "GET %s unreachable (backends running?)", url)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, raw
}

func doJSON(t *testing.T, method, url string, body any) (int, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	var decoded map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&decoded))
	return resp.StatusCode, decoded
}

// createSession reserves a room; returns the public invite code.
func createSession(t *testing.T, host, title, password string) string {
	t.Helper()
	status, body := postJSON(t, signallingBase()+"/session", map[string]any{
		"host": host, "title": title, "password": password,
	})
	require.Equal(t, http.StatusOK, status, "create session: %v", body)
	invite, _ := body["socket"].(string)
	require.NotEmpty(t, invite)
	return invite
}

// joinSession redeems an invite; returns the room socket id for /ws/:socket.
func joinSession(t *testing.T, invite, password string) (title, room string, status int) {
	t.Helper()
	status, body := postJSON(t, signallingBase()+"/connect/"+invite, map[string]any{
		"password": password,
	})
	if status != http.StatusOK {
		return "", "", status
	}
	title, _ = body["title"].(string)
	room, _ = body["socket"].(string)
	require.NotEmpty(t, room)
	return title, room, status
}

// wsMessage mirrors the server envelope {type,userID,description,candidate,to}.
type wsMessage struct {
	Type        string `json:"type"`
	UserID      string `json:"userID"`
	Description string `json:"description,omitempty"`
	Candidate   string `json:"candidate,omitempty"`
	To          string `json:"to,omitempty"`
}

func dialRoom(t *testing.T, room string) *websocket.Conn {
	t.Helper()
	conn, resp, err := websocket.DefaultDialer.Dial(signallingWS()+"/ws/"+room, nil)
	require.NoError(t, err, "dial /ws/%s", room)
	if resp != nil {
		resp.Body.Close()
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func readMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) wsMessage {
	t.Helper()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(timeout)))
	var msg wsMessage
	require.NoError(t, conn.ReadJSON(&msg), "timed out waiting for message")
	return msg
}

// expectNoMessage asserts silence: any arrival within the window fails.
//
// Note: a timed-out gorilla read is sticky — the conn's read side is spent
// afterwards. Only call this on a connection you will not read from again
// (its write side still works).
func expectNoMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) {
	t.Helper()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(timeout)))
	var msg wsMessage
	err := conn.ReadJSON(&msg)
	require.Error(t, err, "expected silence but received %+v", msg)
}

func send(t *testing.T, conn *websocket.Conn, msg wsMessage) {
	t.Helper()
	require.NoError(t, conn.SetWriteDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, conn.WriteJSON(msg))
}

// joinRoom dials and registers userID; drains the session_joined ack plus any
// presence replays so the connection is quiet for the test exchange.
func joinRoom(t *testing.T, room, userID string, expectReplays int) *websocket.Conn {
	t.Helper()
	conn := dialRoom(t, room)
	send(t, conn, wsMessage{Type: "connect", UserID: userID})
	seenAck := false
	for i := 0; i < expectReplays+1; i++ {
		msg := readMessage(t, conn, 10*time.Second)
		if msg.Type == "session_joined" {
			seenAck = true
		} else {
			assert.Equal(t, "connect", msg.Type)
		}
	}
	require.True(t, seenAck, "missing session_joined ack")
	return conn
}

func TestHealthReadinessMetrics(t *testing.T) {
	requireLive(t)
	for name, base := range map[string]string{"signalling": signallingBase(), "users": usersBase()} {
		status, raw := getBody(t, base+"/health")
		require.Equal(t, http.StatusOK, status, "%s /health", name)
		var health map[string]any
		require.NoError(t, json.Unmarshal(raw, &health))
		assert.Equal(t, "healthy", health["status"])

		status, _ = getBody(t, base+"/ready")
		assert.Equal(t, http.StatusOK, status, "%s /ready (mongo reachable)", name)

		status, metrics := getBody(t, base+"/metrics")
		require.Equal(t, http.StatusOK, status, "%s /metrics", name)
		assert.Contains(t, string(metrics), "videoconf_", "%s exposes videoconf_* metrics", name)
	}
}

func TestUsersLifecycle(t *testing.T) {
	requireLive(t)
	name, password := unique("int-user"), "password123"

	// Create.
	status, created := postJSON(t, usersBase()+"/users", map[string]any{
		"name": name, "password": password,
	})
	require.Equal(t, http.StatusCreated, status, "create user: %v", created)
	id, _ := created["id"].(string)
	require.NotEmpty(t, id)
	assert.Equal(t, name, created["name"])
	t.Cleanup(func() { _, _ = doJSON(t, "DELETE", usersBase()+"/users/"+id, nil) })

	// Validation: empty name rejected.
	status, _ = postJSON(t, usersBase()+"/users", map[string]any{"name": "", "password": "x"})
	assert.Equal(t, http.StatusBadRequest, status)

	// Read back.
	status, fetched := doJSON(t, "GET", usersBase()+"/users/"+id, nil)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, name, fetched["name"])

	// Authenticate: token + user shape.
	status, auth := postJSON(t, usersBase()+"/auth", map[string]any{
		"username": name, "password": password,
	})
	require.Equal(t, http.StatusOK, status, "auth: %v", auth)
	token, _ := auth["token"].(string)
	require.NotEmpty(t, token)
	assert.Len(t, strings.Split(token, "."), 3, "token is a JWT")
	user, _ := auth["user"].(map[string]any)
	require.NotNil(t, user)
	assert.Equal(t, id, user["id"])
	assert.Equal(t, name, user["name"])

	// Wrong password and unknown user both 401.
	status, _ = postJSON(t, usersBase()+"/auth", map[string]any{
		"username": name, "password": "wrong",
	})
	assert.Equal(t, http.StatusUnauthorized, status)
	status, _ = postJSON(t, usersBase()+"/auth", map[string]any{
		"username": unique("nobody"), "password": "wrong",
	})
	assert.Equal(t, http.StatusUnauthorized, status)

	// List contains the new user.
	status, raw := getBody(t, usersBase()+"/users")
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(raw), id)

	// Update + read-after-write.
	newName := unique("int-user-renamed")
	status, _ = doJSON(t, "PUT", usersBase()+"/users/"+id, map[string]any{
		"name": newName, "password": password,
	})
	require.Equal(t, http.StatusOK, status)
	status, fetched = doJSON(t, "GET", usersBase()+"/users/"+id, nil)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, newName, fetched["name"])

	// Update of a missing (but well-formed) id is 404.
	status, _ = doJSON(t, "PUT", usersBase()+"/users/000000000000000000000000", map[string]any{
		"name": "x", "password": "y",
	})
	assert.Equal(t, http.StatusNotFound, status)

	// Delete + double-delete 404 + get-after-delete 404.
	status, _ = doJSON(t, "DELETE", usersBase()+"/users/"+id, nil)
	require.Equal(t, http.StatusOK, status)
	status, _ = doJSON(t, "DELETE", usersBase()+"/users/"+id, nil)
	assert.Equal(t, http.StatusNotFound, status)
	status, _ = doJSON(t, "GET", usersBase()+"/users/"+id, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestSessionLifecycle(t *testing.T) {
	requireLive(t)
	host, title, password := unique("host"), "Integration Test Meeting", "meeting123"

	invite := createSession(t, host, title, password)

	// Presence probe.
	status, _ := getBody(t, signallingBase()+"/connect?url="+invite)
	assert.Equal(t, http.StatusOK, status)
	status, _ = getBody(t, signallingBase()+"/connect")
	assert.Equal(t, http.StatusBadRequest, status, "missing url param")

	// Join: title echoes, room socket differs from the invite.
	gotTitle, room, status := joinSession(t, invite, password)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, title, gotTitle)
	assert.NotEqual(t, invite, room)

	// Wrong password, unknown invite, and malformed JSON all 400.
	_, _, status = joinSession(t, invite, "wrong-password")
	assert.Equal(t, http.StatusBadRequest, status)
	_, _, status = joinSession(t, strings.Repeat("0", 64), password)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, http.StatusBadRequest,
		postRaw(t, signallingBase()+"/connect/"+invite, "application/json", "{not-json"))
}

func TestSignallingOfferAnswerFlow(t *testing.T) {
	requireLive(t)
	password := "meeting123"
	invite := createSession(t, unique("host"), "Signalling Flow", password)
	_, room, status := joinSession(t, invite, password)
	require.Equal(t, http.StatusOK, status)

	hostID, guestID := unique("host"), unique("guest")
	host := joinRoom(t, room, hostID, 0)
	guest := joinRoom(t, room, guestID, 1)

	// Host sees the guest's presence broadcast.
	presence := readMessage(t, host, 10*time.Second)
	assert.Equal(t, "connect", presence.Type)
	assert.Equal(t, guestID, presence.UserID)

	// Offer → answer → ICE, all targeted.
	send(t, host, wsMessage{Type: "offer", UserID: hostID, Description: "sdp-offer", To: guestID})
	offer := readMessage(t, guest, 10*time.Second)
	assert.Equal(t, "offer", offer.Type)
	assert.Equal(t, hostID, offer.UserID)
	assert.Equal(t, "sdp-offer", offer.Description)

	send(t, guest, wsMessage{Type: "answer", UserID: guestID, Description: "sdp-answer", To: hostID})
	answer := readMessage(t, host, 10*time.Second)
	assert.Equal(t, "answer", answer.Type)
	assert.Equal(t, guestID, answer.UserID)
	assert.Equal(t, "sdp-answer", answer.Description)

	send(t, host, wsMessage{Type: "ice-candidate", UserID: hostID, Candidate: "cand-1", To: guestID})
	ice := readMessage(t, guest, 10*time.Second)
	assert.Equal(t, "ice-candidate", ice.Type)
	assert.Equal(t, "cand-1", ice.Candidate)

	// Graceful leave broadcasts to the room.
	send(t, guest, wsMessage{Type: "disconnect", UserID: guestID})
	bye := readMessage(t, host, 10*time.Second)
	assert.Equal(t, "disconnect", bye.Type)
	assert.Equal(t, guestID, bye.UserID)
}

func TestUnknownMessageTypeIgnored(t *testing.T) {
	requireLive(t)
	password := "meeting123"
	invite := createSession(t, unique("host"), "Unknown Type", password)
	_, room, status := joinSession(t, invite, password)
	require.Equal(t, http.StatusOK, status)

	aID, bID := unique("a"), unique("b")
	a := joinRoom(t, room, aID, 0)
	b := joinRoom(t, room, bID, 1)
	_ = readMessage(t, a, 10*time.Second) // drain b's presence

	// Unknown types are dropped for forward-compat; the peer hears nothing.
	send(t, a, wsMessage{Type: "broadcast", UserID: aID})
	expectNoMessage(t, b, 2*time.Second)

	// The room still routes afterwards. Direction matters: b's read side is
	// spent by the silence check above, so liveness is proven b→a instead.
	send(t, b, wsMessage{Type: "offer", UserID: bID, Description: "sdp", To: aID})
	msg := readMessage(t, a, 10*time.Second)
	assert.Equal(t, "offer", msg.Type)
	assert.Equal(t, bID, msg.UserID)
}

func TestSpoofedSenderDropped(t *testing.T) {
	requireLive(t)
	password := "meeting123"
	invite := createSession(t, unique("host"), "Spoof", password)
	_, room, status := joinSession(t, invite, password)
	require.Equal(t, http.StatusOK, status)

	aID, bID := unique("a"), unique("b")
	a := joinRoom(t, room, aID, 0)
	b := joinRoom(t, room, bID, 1)
	_ = readMessage(t, a, 10*time.Second) // drain b's presence

	// One connection, one identity: a mismatched sender id is dropped.
	send(t, a, wsMessage{Type: "offer", UserID: "mallory", Description: "sdp", To: bID})
	expectNoMessage(t, b, 2*time.Second)

	// Liveness after the drop, proven b→a (b's read side is spent).
	send(t, b, wsMessage{Type: "offer", UserID: bID, Description: "sdp", To: aID})
	msg := readMessage(t, a, 10*time.Second)
	assert.Equal(t, bID, msg.UserID)
}

func TestRoomIsolation(t *testing.T) {
	requireLive(t)
	password := "meeting123"
	inviteA := createSession(t, unique("host"), "Room A", password)
	inviteB := createSession(t, unique("host"), "Room B", password)
	_, roomA, status := joinSession(t, inviteA, password)
	require.Equal(t, http.StatusOK, status)
	_, roomB, status := joinSession(t, inviteB, password)
	require.Equal(t, http.StatusOK, status)
	require.NotEqual(t, roomA, roomB)

	a1ID, a2ID, b1ID := unique("a1"), unique("a2"), unique("b1")
	a1 := joinRoom(t, roomA, a1ID, 0)
	a2 := joinRoom(t, roomA, a2ID, 1)
	b1 := joinRoom(t, roomB, b1ID, 0)
	_ = readMessage(t, a1, 10*time.Second) // drain a2's presence

	// Untargeted offer stays inside room A.
	send(t, a1, wsMessage{Type: "offer", UserID: a1ID, Description: "sdp"})
	msg := readMessage(t, a2, 10*time.Second)
	assert.Equal(t, "offer", msg.Type)
	assert.Equal(t, a1ID, msg.UserID)
	expectNoMessage(t, b1, 2*time.Second)
}

func TestRoomCapEight(t *testing.T) {
	requireLive(t)
	password := "meeting123"
	invite := createSession(t, unique("host"), "Full Room", password)
	_, room, status := joinSession(t, invite, password)
	require.Equal(t, http.StatusOK, status)

	// Fill the room to the 8-member server cap.
	for i := 0; i < 8; i++ {
		joinRoom(t, room, unique(fmt.Sprintf("member-%d", i)), i)
	}

	// The 9th upgrade is refused with 429.
	_, resp, err := websocket.DefaultDialer.Dial(signallingWS()+"/ws/"+room, nil)
	require.Error(t, err, "9th member should be refused")
	require.NotNil(t, resp)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
}
