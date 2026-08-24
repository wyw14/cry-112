package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wyw14/cry-112/internal/api"
	"github.com/wyw14/cry-112/internal/cycle"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	address := environment("STERICYCLE_ADDR", "127.0.0.1:21212")
	dataDirectory := environment("STERICYCLE_DATA", "data")
	controller, err := cycle.NewController(dataDirectory, time.Now())
	if err != nil {
		return fmt.Errorf("initialize controller: %w", err)
	}
	server := &http.Server{
		Addr:              address,
		Handler:           api.NewServer(controller).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errorsChannel := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errorsChannel <- err
			return
		}
		errorsChannel <- nil
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case signalValue := <-signals:
		log.Printf("received %s, shutting down", signalValue)
		contextValue, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(contextValue); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		return <-errorsChannel
	case err := <-errorsChannel:
		if err != nil {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	}
}

func environment(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
