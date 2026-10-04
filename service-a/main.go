package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"pos-goexpert-desafio7/service-a/app"
)

func main() {
	ctx := context.Background()
	tp, err := app.InitTracer(ctx)
	if err != nil {
		log.Printf("otel init: %v", err)
	}
	if tp != nil {
		defer func() { _ = tp.Shutdown(ctx) }()
	}

	mux := app.NewHandler()
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Println("service-a listening :8080")
	log.Fatal(srv.ListenAndServe())
}
