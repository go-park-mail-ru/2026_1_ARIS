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

	handlers "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/server"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
)

func main() {

	authService := service.NewAuthService()
	jwtSecret := "ключ"
	authHandler := handlers.NewAuthHandler(authService, jwtSecret)

	router := server.NewRouter(authHandler, []byte(jwtSecret))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		fmt.Println("Server is running on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	db := repository.NewRepository()
	db.UserRepo.Save(context.Background(), models.NewUser(1, "KokInside", "KokInside@gmail.com", "+79999999999", "hard_password"))

	users, err := db.UserRepo.List(context.Background(), 0, 1)
	if err != nil {
		fmt.Println("Error listing users:", err)
	} else {
		fmt.Println("Users:", users)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Server is stopping")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	fmt.Println("Server stopped")
}
