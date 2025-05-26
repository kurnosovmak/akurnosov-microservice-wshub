package wshubclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

// Client представляет клиент для работы с WS Hub
type Client struct {
	baseURL    string
	httpClient *http.Client
	conn       *websocket.Conn
	mutex      sync.RWMutex
	handlers   map[string]MessageHandler
	connected  bool
}

// MessageHandler определяет функцию обработчика сообщений
type MessageHandler func([]byte) error

// Config содержит параметры конфигурации клиента
type Config struct {
	BaseURL    string
	HTTPClient *http.Client
}

// CredentialsResponse представляет ответ от метода получения учетных данных
type CredentialsResponse struct {
	Token string `json:"token"`
}

// BroadcastRequest представляет запрос на отправку сообщения
type BroadcastRequest struct {
	Channel string `json:"channel"`
	Message string `json:"message"`
}

// NewClient создает новый экземпляр клиента
func NewClient(config Config) *Client {
	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}

	return &Client{
		baseURL:    config.BaseURL,
		httpClient: config.HTTPClient,
		handlers:   make(map[string]MessageHandler),
	}
}

// GetCredentials получает токен для подключения к определенному каналу
func (c *Client) GetCredentials(channel string) (string, error) {
	reqBody := struct {
		Channel string `json:"channel"`
	}{
		Channel: channel,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/credentials", c.baseURL), bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned non-200 status code: %d", resp.StatusCode)
	}

	var credResp CredentialsResponse
	if err := json.NewDecoder(resp.Body).Decode(&credResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return credResp.Token, nil
}

// Broadcast отправляет сообщение в указанный канал
func (c *Client) Broadcast(channel, message string) error {
	reqBody := BroadcastRequest{
		Channel: channel,
		Message: message,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/broadcast", c.baseURL), bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-200 status code: %d", resp.StatusCode)
	}

	return nil
}

// Connect устанавливает WebSocket соединение
func (c *Client) Connect(channel, token string) error {
	if c.connected {
		return fmt.Errorf("already connected")
	}

	// Формируем URL для подключения
	u := url.URL{
		Scheme: "ws",
		Host:   c.baseURL,
		Path:   "/ws",
	}

	// Добавляем параметры запроса
	q := u.Query()
	q.Set("token", token)
	q.Set("channel", channel)
	u.RawQuery = q.Encode()

	// Устанавливаем соединение
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}

	c.conn = conn
	c.connected = true
	return nil
}

// Close закрывает WebSocket соединение
func (c *Client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		c.connected = false
		return c.conn.Close()
	}
	return nil
}

// AddHandler добавляет обработчик для сообщений
func (c *Client) AddHandler(messageType string, handler MessageHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.handlers[messageType] = handler
}

// Listen начинает прослушивание входящих сообщений
func (c *Client) Listen() error {
	if !c.connected || c.conn == nil {
		return fmt.Errorf("not connected")
	}

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("error reading message: %w", err)
		}

		// Вызываем обработчик для полученного сообщения
		if handler, ok := c.handlers["default"]; ok {
			if err := handler(message); err != nil {
				return fmt.Errorf("handler error: %w", err)
			}
		}
	}
}
