package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var cepRegex = regexp.MustCompile(`^[0-9]{8}$`)

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/weather", weatherHandler)
	mux.HandleFunc("/healthz", healthHandler)
	return mux
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	// extract incoming context for tracing
	prop := otel.GetTextMapPropagator()
	ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	tracer := otel.Tracer("service-a")
	ctx, span := tracer.Start(ctx, "input-weather")
	defer span.End()
	var payload struct {
		Cep string `json:"cep"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid zipcode", 422)
		return
	}
	cep := payload.Cep
	if !cepRegex.MatchString(cep) {
		http.Error(w, "invalid zipcode", 422)
		return
	}

	// forward to service-b
	svc := os.Getenv("SERVICE_B_URL")
	if svc == "" {
		svc = "http://service-b:8081/weather"
	}
	client := http.Client{Timeout: 10 * time.Second}
	body, _ := json.Marshal(map[string]string{"cep": cep})
	// record request body as attribute (be cautious with sensitive data)
	span.SetAttributes(attribute.String("http.request.body", string(body)))
	req, _ := http.NewRequestWithContext(ctx, "POST", svc, io.NopCloser(bytes.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	if ct := req.Header.Get("Content-Type"); ct != "" {
		span.SetAttributes(attribute.String("http.request.header.content_type", ct))
	}
	// inject tracing headers to propagate trace to service-b
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	sc := trace.SpanContextFromContext(ctx)
	log.Printf("service-a: sending request to %s trace=%s headers=%v", svc, sc.TraceID().String(), req.Header)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error calling service-b: %v", err)
		http.Error(w, "can not get weather", 500)
		return
	}
	defer resp.Body.Close()
	// proxy response codes and body
	b, _ := io.ReadAll(resp.Body)
	// record response status and body
	span.SetAttributes(attribute.Int("http.response.status_code", resp.StatusCode))
	span.AddEvent("http.response.body", trace.WithAttributes(attribute.String("body", string(b))))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(b)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
