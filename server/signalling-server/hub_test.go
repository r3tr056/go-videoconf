package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/r3tr056/go-videoconf/signalling-server/controllers"
	"github.com/r3tr056/go-videoconf/signalling-server/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveWS mounts the real wshandler for one socket path on a test server,
// backed by an isolated relay it returns for assertions.
func serveWS(t *testing.T, socket string) (*httptest.Server, *MeshRelay) {
	t.Helper()
	relay := NewMeshRelay()
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/"+socket, func(w http.ResponseWriter, r *http.Request) {
		wshandler(w, r, socket, relay)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, relay
}

func dialWS(t *testing.T, server *httptest.Server, socket string, header http.Header) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/" + socket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	return conn
}

func readMessage(t *testing.T, conn *websocket.Conn) interfaces.Message {
	t.Helper()
	var msg interfaces.Message
	require.NoError(t, conn.ReadJSON(&msg))
	return msg
}

// liveConnPair returns a (server-side, client-side) connection pair over a
// throwaway test server. Relay removal closes the socket, so tests need real
// connections rather than zero structs.
func liveConnPair(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	got := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		got <- conn
	}))
	t.Cleanup(server.Close)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })
	return <-got, client
}

func TestMeshRelayAddRemoveMembers(t *testing.T) {
	r := NewMeshRelay()
	assert.Empty(t, r.Members("room"))

	srvA, _ := liveConnPair(t)
	srvB, _ := liveConnPair(t)
	r.Add("room", "a", srvA)
	r.Add("room", "b", srvB)
	assert.ElementsMatch(t, []string{"a", "b"}, r.Members("room"))

	// Re-adding replaces the handle.
	srvA2, _ := liveConnPair(t)
	r.Add("room", "a", srvA2)
	assert.ElementsMatch(t, []string{"a", "b"}, r.Members("room"))

	r.Remove("room", "a")
	assert.Equal(t, []string{"b"}, r.Members("room"))

	// Removing the last member drops the room.
	r.Remove("room", "b")
	assert.Empty(t, r.Members("room"))

	// Removing unknown members/rooms is a no-op.
	r.Remove("room", "ghost")
	r.Remove("nope", "ghost")
}

func TestWSHandlerPresenceBothDirections(t *testing.T) {
	server, relay := serveWS(t, "presence-room")

	a := dialWS(t, server, "presence-room", nil)
	require.NoError(t, a.WriteJSON(interfaces.Message{Type: "connect", UserID: "a"}))
	ack := readMessage(t, a)
	assert.Equal(t, "session_joined", ack.Type)
	assert.Equal(t, "a", ack.UserID)

	b := dialWS(t, server, "presence-room", nil)
	require.NoError(t, b.WriteJSON(interfaces.Message{Type: "connect", UserID: "b"}))

	// Existing member learns about the newcomer.
	joined := readMessage(t, a)
	assert.Equal(t, "connect", joined.Type)
	assert.Equal(t, "b", joined.UserID)

	// Newcomer replays existing members, then gets its ack.
	replay := readMessage(t, b)
	assert.Equal(t, "connect", replay.Type)
	assert.Equal(t, "a", replay.UserID)
	back := readMessage(t, b)
	assert.Equal(t, "session_joined", back.Type)

	// Offers relay to the other member.
	require.NoError(t, b.WriteJSON(interfaces.Message{
		Type: "offer", UserID: "b", Description: "sdp", To: "a",
	}))
	offer := readMessage(t, a)
	assert.Equal(t, "offer", offer.Type)
	assert.Equal(t, "b", offer.UserID)

	// Graceful disconnect notifies the room and prunes the member.
	require.NoError(t, a.WriteJSON(interfaces.Message{Type: "disconnect", UserID: "a"}))
	bye := readMessage(t, b)
	assert.Equal(t, "disconnect", bye.Type)
	require.Eventually(t, func() bool {
		members := relay.Members("presence-room")
		return len(members) == 1 && members[0] == "b"
	}, 5*time.Second, 10*time.Millisecond)
}

func TestWSHandlerTargetedDelivery(t *testing.T) {
	server, _ := serveWS(t, "target-room")

	join := func(user string) *websocket.Conn {
		c := dialWS(t, server, "target-room", nil)
		require.NoError(t, c.WriteJSON(interfaces.Message{Type: "connect", UserID: user}))
		// Drain replays until the ack.
		for {
			if msg := readMessage(t, c); msg.Type == "session_joined" {
				break
			}
		}
		return c
	}

	a := join("a")
	b := join("b")
	c := join("c")

	// Drain the join notifications a and b accumulated.
	assert.Equal(t, "b", readMessage(t, a).UserID) // b joined
	assert.Equal(t, "c", readMessage(t, a).UserID) // c joined
	assert.Equal(t, "c", readMessage(t, b).UserID) // c joined

	// Targeted offer reaches its destination.
	require.NoError(t, a.WriteJSON(interfaces.Message{
		Type: "offer", UserID: "a", Description: "sdp", To: "b",
	}))
	got := readMessage(t, b)
	assert.Equal(t, "offer", got.Type)
	assert.Equal(t, "a", got.UserID)

	// Untargeted messages still broadcast to the whole room.
	require.NoError(t, a.WriteJSON(interfaces.Message{
		Type: "offer", UserID: "a", Description: "sdp",
	}))
	assert.Equal(t, "offer", readMessage(t, b).Type)
	assert.Equal(t, "offer", readMessage(t, c).Type)

	// Targeted messages never leak to bystanders (negative check last: a
	// timed-out client read is not reused afterwards).
	require.NoError(t, b.WriteJSON(interfaces.Message{
		Type: "answer", UserID: "b", Description: "sdp", To: "a",
	}))
	assert.Equal(t, "answer", readMessage(t, a).Type)
	require.NoError(t, c.SetReadDeadline(time.Now().Add(300*time.Millisecond)))
	var leaked interfaces.Message
	require.Error(t, c.ReadJSON(&leaked), "targeted message must not leak to bystanders")
}

func TestWSHandlerRoomFull(t *testing.T) {
	server, relay := serveWS(t, "full-room")

	conns := make([]*websocket.Conn, 0, maxMembersPerSocket)
	for i := 0; i < maxMembersPerSocket; i++ {
		user := strings.Repeat("u", 1) + string(rune('a'+i))
		c := dialWS(t, server, "full-room", nil)
		require.NoError(t, c.WriteJSON(interfaces.Message{Type: "connect", UserID: user}))
		// Drain until the ack arrives (replays come first for later joiners).
		for {
			msg := readMessage(t, c)
			if msg.Type == "session_joined" {
				break
			}
		}
		conns = append(conns, c)
	}
	require.Len(t, relay.Members("full-room"), maxMembersPerSocket)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/full-room"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.Error(t, err)
	if resp != nil {
		resp.Body.Close()
		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	}
}

func TestWSHandlerOriginAllowlist(t *testing.T) {
	t.Setenv("WS_ALLOWED_ORIGINS", "https://app.example.com")
	server, _ := serveWS(t, "origin-room")

	// No Origin header: rejected.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/origin-room"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.Error(t, err)
	if resp != nil {
		resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}

	// Wrong origin: rejected.
	_, resp, err = websocket.DefaultDialer.Dial(wsURL, http.Header{"Origin": {"https://evil.example.com"}})
	require.Error(t, err)
	if resp != nil {
		resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}

	// Allowed origin: accepted.
	ok := dialWS(t, server, "origin-room", http.Header{"Origin": {"https://app.example.com"}})
	require.NoError(t, ok.WriteJSON(interfaces.Message{Type: "connect", UserID: "member"}))
	assert.Equal(t, "session_joined", readMessage(t, ok).Type)
}

func TestWSHandlerRejectsEmptyUserID(t *testing.T) {
	server, relay := serveWS(t, "noid-room")
	conn := dialWS(t, server, "noid-room", nil)
	require.NoError(t, conn.WriteJSON(interfaces.Message{Type: "connect", UserID: ""}))
	var msg interfaces.Message
	require.Error(t, conn.ReadJSON(&msg), "server must close connections with an empty user id")
	assert.Empty(t, relay.Members("noid-room"))
}

func TestWSHandlerDropsSpoofedSender(t *testing.T) {
	server, _ := serveWS(t, "spoof-room")
	a := dialWS(t, server, "spoof-room", nil)
	require.NoError(t, a.WriteJSON(interfaces.Message{Type: "connect", UserID: "a"}))
	assert.Equal(t, "session_joined", readMessage(t, a).Type)

	b := dialWS(t, server, "spoof-room", nil)
	require.NoError(t, b.WriteJSON(interfaces.Message{Type: "connect", UserID: "b"}))
	assert.Equal(t, "connect", readMessage(t, a).Type) // a sees b join
	assert.Equal(t, "connect", readMessage(t, b).Type) // b replays a
	assert.Equal(t, "session_joined", readMessage(t, b).Type)

	// a tries to speak as b: the message must not reach b.
	require.NoError(t, a.WriteJSON(interfaces.Message{Type: "offer", UserID: "b", Description: "x"}))
	require.NoError(t, b.SetReadDeadline(time.Now().Add(300*time.Millisecond)))
	var msg interfaces.Message
	require.Error(t, b.ReadJSON(&msg), "spoofed message must be dropped")
}

func TestRateLimiter(t *testing.T) {
	l := newRateLimiter(3, 50*time.Millisecond)
	assert.True(t, l.allow())
	assert.True(t, l.allow())
	assert.True(t, l.allow())
	assert.False(t, l.allow())
	time.Sleep(60 * time.Millisecond)
	assert.True(t, l.allow())
}

func TestParseOrigins(t *testing.T) {
	assert.Empty(t, parseOrigins(""))
	assert.Empty(t, parseOrigins("  , "))
	assert.Equal(t,
		[]string{"https://a.example", "https://b.example"},
		parseOrigins("https://a.example, https://b.example,, "))
}

func TestGetSessionMissingParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/connect", controllers.GetSession)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/connect", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
