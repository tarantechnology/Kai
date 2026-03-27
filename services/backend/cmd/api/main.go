package main

import (
	"log"
	"net/http"

	"github.com/taran/kai/services/backend/internal/config"
	"github.com/taran/kai/services/backend/internal/server"
)

func main() {
	cfg := config.Load()

	log.Printf("kai backend listening on :%s", cfg.Port)

	if err := http.ListenAndServe(":"+cfg.Port, server.New()); err != nil {
		log.Fatal(err)
	}
}
