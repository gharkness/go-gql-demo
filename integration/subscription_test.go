package integration_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gharkness/go-gql-demo/internal/auth"
	"github.com/gharkness/go-gql-demo/internal/reviews"
	"github.com/gorilla/websocket"
)

func TestSubscriptionReceivesCreatedReview(t *testing.T) {
	s := httptest.NewServer(reviews.Handler(database(t), auth.New("http://localhost:8085/realms/bookstore", "http://localhost:8085/realms/bookstore/protocol/openid-connect/certs").Middleware))
	defer s.Close()
	subscriptionRoundTrip(t, s.URL, s.URL)
}
func subscriptionRoundTrip(t *testing.T, httpURL, wsURL string) {
	t.Helper()
	accessToken := token(t)
	dialer := websocket.Dialer{Subprotocols: []string{"graphql-transport-ws"}}
	conn, _, err := dialer.Dial("ws"+strings.TrimPrefix(wsURL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteJSON(map[string]any{"type": "connection_init"}); err != nil {
		t.Fatal(err)
	}
	var ack map[string]any
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatal(err)
	}
	if ack["type"] != "connection_ack" {
		t.Fatalf("ack: %v", ack)
	}
	conn.WriteJSON(map[string]any{"id": "review-events", "type": "subscribe", "payload": map[string]string{"query": `subscription { reviewCreated(bookId:"1") { id body book { id } } }`}})
	messages := make(chan map[string]any, 1)
	failures := make(chan error, 1)
	go func() {
		var msg map[string]any
		if err := conn.ReadJSON(&msg); err != nil {
			failures <- err
			return
		}
		messages <- msg
	}()
	// The protocol has no per-operation ready acknowledgement. Keep producing events
	// until the subscriber is registered, instead of depending on an arbitrary sleep.
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(8 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case <-ticker.C:
			result := graphqlRequest(t, httpURL, createReview, accessToken)
			if result["errors"] != nil {
				t.Fatalf("mutation: %v", result)
			}
		case msg := <-messages:
			if msg["type"] != "next" {
				b, _ := json.Marshal(msg)
				t.Fatalf("expected event: %s", b)
			}
			payload := msg["payload"].(map[string]any)
			if payload["errors"] != nil {
				t.Fatalf("subscription: %v", payload)
			}
			event := payload["data"].(map[string]any)["reviewCreated"].(map[string]any)
			if event["body"] != "Helpful example" || event["book"].(map[string]any)["id"] != "1" {
				t.Fatalf("event: %v", event)
			}
			conn.WriteJSON(map[string]string{"id": "review-events", "type": "complete"})
			return
		case err := <-failures:
			t.Fatal(err)
		case <-deadline.C:
			t.Fatal("subscription timed out")
		}
	}
}
