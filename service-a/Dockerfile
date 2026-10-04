FROM golang:1.20-alpine AS build
WORKDIR /app
COPY . .
RUN apk add --no-cache git
RUN go get go.opentelemetry.io/otel@v1.14.0 go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.14.0 go.opentelemetry.io/otel/sdk@v1.14.0 || true
RUN go mod download
RUN go build -o /service-a ./

FROM alpine:3.18
COPY --from=build /service-a /service-a
EXPOSE 8080
ENTRYPOINT ["/service-a"]
