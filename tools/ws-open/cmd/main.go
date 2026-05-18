package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	baseURL := flag.String("base-url", "ws://localhost:18080", "websocket base URL")
	chatID := flag.Int64("chat-id", 1, "chat ID")
	cookieFile := flag.String("cookie-file", "/tmp/aris-cookies.txt", "curl cookie jar with session_id")
	duration := flag.Duration("duration", 0, "hold connection for duration; 0 means until Ctrl+C")
	flag.Parse()

	sessionID := strings.TrimSpace(os.Getenv("SESSION_ID"))
	if sessionID == "" {
		var err error
		sessionID, err = sessionIDFromCookieFile(*cookieFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	wsURL, err := websocketURL(*baseURL, *chatID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	header := http.Header{}
	header.Set("Cookie", "session_id="+sessionID)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open websocket: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("websocket opened: %s\n", wsURL)
	if *duration == 0 {
		fmt.Println("press Ctrl+C to close it")
	} else {
		fmt.Printf("will close after %s\n", duration.String())
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	if *duration > 0 {
		timer := time.NewTimer(*duration)
		defer timer.Stop()
		select {
		case <-done:
		case <-timer.C:
		case <-stop:
		}
	} else {
		select {
		case <-done:
		case <-stop:
		}
	}
}

func websocketURL(base string, chatID int64) (string, error) {
	if chatID <= 0 {
		return "", fmt.Errorf("chat ID must be positive")
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported websocket URL scheme: %s", parsed.Scheme)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/ws/" + strconv.FormatInt(chatID, 10)
	return parsed.String(), nil
}

func sessionIDFromCookieFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("session cookie not found: run auth login first or set SESSION_ID")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 7 && fields[len(fields)-2] == "session_id" {
			return fields[len(fields)-1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("session_id not found in %s: run auth login again", path)
}
