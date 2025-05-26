package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kurnosovmak/akurnosov-microservice-wshub/internal/config"
	"github.com/kurnosovmak/akurnosov-microservice-wshub/internal/handlers"
	"github.com/kurnosovmak/akurnosov-microservice-wshub/internal/middleware"

	"github.com/gorilla/mux"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	h := handlers.NewHandler(cfg.JWT)

	// Настраиваем маршруты
	r := mux.NewRouter()

	r.Use()

	r.Handle("/ws", middleware.CORSMiddleware(middleware.JWTMiddleware(cfg.JWT)(http.HandlerFunc(h.WS))))
	r.HandleFunc("/broadcast", h.Broadcast).Methods("POST")
	r.HandleFunc("/credentials", h.ConnectionCredentials).Methods("POST")

	// Настраиваем сервер
	serverAddr := fmt.Sprintf(":%s", cfg.HTTP.Port)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запускаем сервер в горутине
	go func() {
		log.Printf("Server listening on %s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Настраиваем graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
