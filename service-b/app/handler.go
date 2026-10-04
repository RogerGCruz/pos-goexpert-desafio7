package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var cepRegex = regexp.MustCompile(`^[0-9]{8}$`)

type WeatherResp struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/weather", weatherHandler)
	mux.HandleFunc("/healthz", healthHandler)
	return mux
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	// log incoming headers and extract incoming context for distributed tracing
	log.Printf("service-b: incoming headers=%v", r.Header)
	prop := otel.GetTextMapPropagator()
	ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	tracer := otel.Tracer("service-b")
	ctx, span := tracer.Start(ctx, "weather")
	defer span.End()
	sc := trace.SpanContextFromContext(ctx)
	log.Printf("service-b: extracted trace=%s", sc.TraceID().String())

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
	// add request attributes to span
	span.SetAttributes(attribute.String("request.cep", cep))
	if ct := r.Header.Get("Content-Type"); ct != "" {
		span.SetAttributes(attribute.String("http.request.header.content_type", ct))
	}
	if ua := r.Header.Get("User-Agent"); ua != "" {
		span.SetAttributes(attribute.String("http.request.header.user_agent", ua))
	}

	city, err := lookupCityByCEP(ctx, cep)
	if err != nil {
		if errors.Is(err, errNotFound) {
			http.Error(w, "can not find zipcode", 404)
			return
		}
		http.Error(w, "can not find zipcode", 404)
		return
	}

	tempC, err := lookupTemperatureByCity(ctx, city)
	if err != nil {
		http.Error(w, "can not get weather", 500)
		return
	}

	resp := WeatherResp{
		City:  city,
		TempC: round(tempC, 2),
		TempF: round(cToF(tempC), 2),
		TempK: round(cToK(tempC), 2),
	}
	// enrich span with response details
	span.SetAttributes(
		attribute.String("response.city", resp.City),
		attribute.Float64("response.temp_C", resp.TempC),
		attribute.Float64("response.temp_F", resp.TempF),
		attribute.Float64("response.temp_K", resp.TempK),
	)
	span.AddEvent("response.payload", trace.WithAttributes(attribute.String("body", func() string {
		b, _ := json.Marshal(resp)
		return string(b)
	}())))
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

var errNotFound = errors.New("not found")

func lookupCityByCEP(ctx context.Context, cep string) (string, error) {
	tracer := otel.Tracer("service-b")
	ctx, span := tracer.Start(ctx, "lookup_city")
	defer span.End()
	span.SetAttributes(attribute.String("cep", cep))

	base := os.Getenv("VIACEP_BASE_URL")
	if base == "" {
		base = "https://viacep.com.br"
	}
	full := base + "/ws/" + cep + "/json"
	client := http.Client{Timeout: 5 * time.Second}
	var resp *http.Response
	var err error
	// simple retry
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequestWithContext(ctx, "GET", full, nil)
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
		resp, err = client.Do(req)
		if err == nil {
			break
		}
		span.RecordError(err)
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("viacep error")
	}

	var data struct {
		Localidade string `json:"localidade"`
		Erro       bool   `json:"erro"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.Erro || data.Localidade == "" {
		return "", errNotFound
	}
	return data.Localidade, nil
}

func lookupTemperatureByCity(ctx context.Context, city string) (float64, error) {
	tracer := otel.Tracer("service-b")
	ctx, span := tracer.Start(ctx, "lookup_temperature")
	defer span.End()
	span.SetAttributes(attribute.String("city", city))

	apiKey := os.Getenv("WEATHERAPI_KEY")
	if apiKey == "" {
		return 0, errors.New("missing WEATHERAPI_KEY")
	}
	base := os.Getenv("WEATHERAPI_BASE_URL")
	if base == "" {
		base = "http://api.weatherapi.com"
	}
	q := url.QueryEscape(city)
	full := base + "/v1/current.json?key=" + apiKey + "&q=" + q + "&aqi=no"
	client := http.Client{Timeout: 7 * time.Second}
	var resp *http.Response
	var err error
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequestWithContext(ctx, "GET", full, nil)
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
		resp, err = client.Do(req)
		if err == nil {
			break
		}
		span.RecordError(err)
		time.Sleep(150 * time.Millisecond)
	}
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		log.Printf("weatherapi status=%d url=%s body=%s", resp.StatusCode, full, string(b))
		return 0, errors.New("weatherapi error")
	}
	var data struct {
		Current struct {
			TempC float64 `json:"temp_c"`
		} `json:"current"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}
	return data.Current.TempC, nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func round(v float64, prec int) float64 {
	p := 1.0
	for i := 0; i < prec; i++ {
		p *= 10
	}
	return float64(int(v*p+0.5)) / p
}

func cToF(c float64) float64 { return c*1.8 + 32 }
func cToK(c float64) float64 { return c + 273 }
