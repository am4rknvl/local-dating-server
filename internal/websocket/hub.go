package websocket

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

type Client struct {
	hub            *Hub
	conn           *websocket.Conn
	send           chan []byte
	userID         uint
	conversationID uint
}

type Message struct {
	Type           string `json:"type"`
	ConversationID uint   `json:"conversation_id"`
	SenderID       uint   `json:"sender_id"`
	Content        string `json:"content"`
	MessageType    string `json:"message_type"`
	Timestamp      string `json:"timestamp"`
}

type TypingMessage struct {
	Type           string `json:"type"`
	ConversationID uint   `json:"conversation_id"`
	UserID         uint   `json:"user_id"`
	IsTyping       bool   `json:"is_typing"`
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("Client connected: User ID %d", client.userID)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client disconnected: User ID %d", client.userID)
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Hub) BroadcastToConversation(conversationID uint, message []byte) {
	for client := range h.clients {
		if client.conversationID == conversationID {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

func (h *Hub) BroadcastToUser(userID uint, message []byte) {
	for client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

func HandleWebSocket(hub *Hub, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		conn.Close()
		return
	}

	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID.(uint),
	}

	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse message to determine type and conversation
		var message map[string]interface{}
		if err := json.Unmarshal(messageBytes, &message); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		// Handle different message types
		switch message["type"] {
		case "join_conversation":
			if convID, ok := message["conversation_id"].(float64); ok {
				c.conversationID = uint(convID)
			}
		case "typing":
			// Broadcast typing indicator to conversation participants
			if convID, ok := message["conversation_id"].(float64); ok {
				typingMsg := TypingMessage{
					Type:           "typing",
					ConversationID: uint(convID),
					UserID:         c.userID,
					IsTyping:       true,
				}
				if msgBytes, err := json.Marshal(typingMsg); err == nil {
					c.hub.BroadcastToConversation(uint(convID), msgBytes)
				}
			}
		case "stop_typing":
			if convID, ok := message["conversation_id"].(float64); ok {
				typingMsg := TypingMessage{
					Type:           "typing",
					ConversationID: uint(convID),
					UserID:         c.userID,
					IsTyping:       false,
				}
				if msgBytes, err := json.Marshal(typingMsg); err == nil {
					c.hub.BroadcastToConversation(uint(convID), msgBytes)
				}
			}
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}
}
