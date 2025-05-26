package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kurnosovmak/akurnosov-microservice-wshub/internal/config"
	"github.com/kurnosovmak/akurnosov-microservice-wshub/internal/middleware"

	"github.com/golang-jwt/jwt/v5"
)

type Handler struct {
	config config.JWTConfig
	m      *sync.RWMutex
	cons   map[string][]*websocket.Conn
}

func NewHandler(config config.JWTConfig) *Handler {
	return &Handler{
		config: config,
		m:      &sync.RWMutex{},
		cons:   make(map[string][]*websocket.Conn),
	}
}

func getChannel(r *http.Request) (string, bool) {
	channel, ok := r.Context().Value(middleware.ChannelKey).(string)
	return channel, ok
}

func (h *Handler) WS(w http.ResponseWriter, r *http.Request) {
	log.Println("Http connected")
	channel, ok := getChannel(r)
	if !ok {
		http.Error(w, "Invalid channel or token", http.StatusBadRequest)
		return
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development, configure appropriately for production
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not upgrade to websocket connection", http.StatusInternalServerError)
		return
	}
	go func(con *websocket.Conn, ctx context.Context) {
		log.Println("WS connected")
		h.m.Lock()
		h.cons[channel] = append(h.cons[channel], con)
		h.m.Unlock()
		defer func() {
			h.m.Lock()
			defer h.m.Unlock()
			// Remove the closed connection from the slice
			newConns := make([]*websocket.Conn, 0, len(h.cons[channel])-1)
			for _, c := range h.cons[channel] {
				if c != con {
					newConns = append(newConns, c)
				}
			}
			h.cons[channel] = newConns
			conn.Close()

		}()
		<-ctx.Done()
	}(conn, r.Context())
}

func (h *Handler) Broadcast(w http.ResponseWriter, r *http.Request) {
	req := &struct {
		Channel string `json:"channel"`
		Message string `json:"message"`
	}{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil || req.Channel == "" || req.Message == "" {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	channel := req.Channel
	message := []byte(req.Message)
	w.WriteHeader(http.StatusOK) // Возвращаем успешный статус
	go func(channel string, message []byte) {
		h.m.RLock()
		defer h.m.RUnlock()
		for _, conn := range h.cons[channel] {
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error broadcasting message: %v\n", err)
			}
		}
	}(channel, message)
}

func (h *Handler) ConnectionCredentials(w http.ResponseWriter, r *http.Request) {
	req := &struct {
		Channel string `json:"channel"`
	}{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil || req.Channel == "" {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	channel := req.Channel
	token, err := generateConnectionToken(channel, h.config.TTL, h.config)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	resp := &struct {
		Token string `json:"token"`
	}{
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func generateConnectionToken(channel string, ttl int64, config config.JWTConfig) (string, error) {
	// Генерация JWT токена
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		middleware.ChannelKey: channel,
		"exp":                 time.Now().Add(time.Duration(ttl) * time.Second).Unix(),
	})

	return token.SignedString([]byte(config.Secret))
}
