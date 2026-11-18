package main

import (
	"log"
	"net/http"

	"github.com/alexfishkazzz/shortener/internal/app"
)

func main() {
	application := app.New()

	log.Println("Запуск сервера на :8080")
	if err := http.ListenAndServe(":8080", application); err != nil {
		log.Fatal(err)
	}
}
