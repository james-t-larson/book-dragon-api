package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "book-dragon/docs"
	"book-dragon/internal/auth"
	"book-dragon/internal/handlers"
	appmiddleware "book-dragon/internal/middleware"
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
	dragonHandler := &handlers.DragonHandler{Store: st}
	bookHandler := &handlers.BookHandler{Store: st}
	tourneyHandler := &handlers.TourneyHandler{Store: st}

	r := chi.NewRouter()

	// Generic middlewares
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
		MaxAge:         300,
	}))
	r.Use(appmiddleware.RequestLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json"))

	// Public routes
	r.Post("/register", userHandler.Register)
	r.Post("/login", userHandler.Login)
	r.Get("/constants", tourneyHandler.GetConstants)

	// Swagger documentation route
	r.Get("/swagger/*", httpSwagger.Handler())

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware)
		r.Get("/auth/me", userHandler.Me)
		r.Post("/logout", userHandler.Logout)
		r.Post("/focus_timer_complete", userHandler.FocusTimerComplete)

		// Dragon routes
		r.Post("/dragon", dragonHandler.CreateDragon)
		r.Get("/dragon", dragonHandler.GetDragon)

		// Book routes
		r.Post("/books", bookHandler.PostBook)
		r.Get("/books", bookHandler.GetBooks)
		r.Put("/books/{id}", bookHandler.UpdateBook)

		// Tourney routes
		r.Get("/tourney", tourneyHandler.GetTourney)
		r.Post("/tourney", tourneyHandler.CreateTourney)
		r.Post("/join_tourney", tourneyHandler.JoinTourney)
	})

	port := "8080"
	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
