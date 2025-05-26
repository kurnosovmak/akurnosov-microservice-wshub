package middleware

import (
	"context"
	"log"
	"net/http"

	"github.com/golang-jwt/jwt/v5"

	"github.com/kurnosovmak/akurnosov-microservice-wshub/internal/config"
)

const (
	ChannelKey = "channel"
)

func JWTMiddleware(cfg config.JWTConfig) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := r.URL.Query().Get("token")
			if tokenStr == "" {
				http.Error(w, "Missing token", http.StatusUnauthorized)
				return
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				return []byte(cfg.Secret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				log.Println(err)
				return
			}

			claims := token.Claims.(jwt.MapClaims)
			channel, ok := claims["channel"].(string)
			if !ok {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				log.Println("Invalid token claims")
				return
			}

			ctx := context.WithValue(r.Context(), ChannelKey, channel)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CORSMiddleware добавляет заголовки CORS
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
