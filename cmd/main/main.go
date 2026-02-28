package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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
