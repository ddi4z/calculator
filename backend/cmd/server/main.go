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
	if err := http.ListenAndServe(addr, calculator.NewHandler(calculator.NewCalculator())); err != nil {
		log.Fatal(err)
	}
}
