package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"pos-goexpert-desafio7/service-b/app"
)

func main() {
	// init tracer
	ctx := context.Background()
	tp, err := app.InitTracer(ctx)
	if err != nil {
		log.Printf("otel init error: %v", err)
	}
	if tp != nil {
		defer func() { _ = tp.Shutdown(ctx) }()
	}

	mux := app.NewHandler()
	srv := &http.Server{
		Addr:         ":8081",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Println("service-b listening :8081")
	log.Fatal(srv.ListenAndServe())
}
