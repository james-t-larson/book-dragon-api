package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "book-dragon/docs"
	"book-dragon/internal/auth"
	"book-dragon/internal/handlers"
	"book-dragon/internal/store"
)

// @title Book Dragon API
// @version 1.0
// @description REST API for the Book Dragon project.
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	dbPath := "bookdragon.db"
	st, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	userHandler := &handlers.UserHandler{Store: st}

	r := chi.NewRouter()

	// Generic middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json"))

	// Public routes
	r.Post("/register", userHandler.Register)
	r.Post("/login", userHandler.Login)

	// Swagger documentation route
	r.Get("/swagger/*", httpSwagger.Handler())

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware)
		r.Get("/auth/me", userHandler.Me)
	})

	port := "8080"
	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
