package main

import (
	"log"

	"backend/internal/router"
)

func main() {
	r := router.New()

	log.Println("server listening on :8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatal(err)
	}
}
