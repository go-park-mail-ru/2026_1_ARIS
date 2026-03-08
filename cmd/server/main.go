package main

import (
	"log"
	"net/http"

	handlers "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/server"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
)

func main() {

	authService := service.NewAuthService()

	jwtSecret := "ключ"
	authHandler := handlers.NewAuthHandler(authService, jwtSecret)

	router := server.NewRouter(authHandler, []byte(jwtSecret))

	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}
