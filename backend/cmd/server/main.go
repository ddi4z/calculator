package main

import (
	"log"
	"net/http"
	"os"

	"calculator/backend/internal/calculator"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("calculator backend listening on %s", addr)
	server := &http.Server{
		Addr:    addr,
		Handler: calculator.NewHandler(calculator.NewCalculator()),
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
