package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello world")
	})

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		fmt.Println("Server started")
		fmt.Println("Server is running on http://localhost:8080")

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Println("Internal server error", err)
			return
		}
	}()

	// инициализация репозитория и добавление тестовых данных
	db := repository.NewRepository()

	db.UserRepo.Save(context.Background(), model.NewUser(1, "KokInside", "KokInside@gmail.com", "+79999999999", "hard_password"))

	users, err := db.UserRepo.List(context.Background(), 0, 1)
	if err != nil {
		fmt.Println("Error listing users:", err)
	} else {
		fmt.Println("Users:", users)
	}

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)

	<-stop

	fmt.Println("Server is stopping")

	ctx, cansel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cansel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Println(err)
	}

	fmt.Println("Server stopped")
}
