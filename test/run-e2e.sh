#!/usr/bin/env bash
set -euo pipefail

# Script para levantar o ambiente de integração (WireMock, services, collector, zipkin),
# executar os testes BDD (Godog) e aguardar uma tecla para derrubar o ambiente criado.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "Run E2E: preparing environment"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker not found in PATH"
  exit 1
fi

# WEATHERAPI_KEY must be provided by the developer (not committed).
if [ -z "${WEATHERAPI_KEY:-}" ]; then
  echo "WEATHERAPI_KEY not defined in environment." >&2
  read -s -p "Enter WEATHERAPI_KEY (input hidden): " WEATHERAPI_KEY
  echo
fi

export VIACEP_BASE_URL='http://wiremock:8080'
export WEATHERAPI_BASE_URL='http://wiremock:8080'
export WEATHERAPI_KEY

echo "Starting docker compose (this may take a while)..."
docker compose up --build -d

echo "Waiting for services health endpoints..."
ready=false
for i in {1..30}; do
  if curl -sSf http://localhost:8080/healthz >/dev/null 2>&1 && curl -sSf http://localhost:8081/healthz >/dev/null 2>&1; then
    ready=true
    break
  fi
  sleep 1
done

if [ "$ready" != true ]; then
  echo "Services did not become healthy in time. Showing logs:" >&2
  docker compose logs --tail=200
  echo "You can inspect logs above. Press any key to teardown." 
  read -n1 -r -p "Press any key to teardown..." dummy
  docker compose down --volumes
  exit 1
fi

echo "Services healthy — running BDD tests (Godog)"
(
  cd test
  go test -v
) || echo "BDD tests failed (see output above)"

echo
read -n1 -r -p "Press any key to teardown the docker environment..." dummy

echo "Tearing down"
docker compose down --volumes

echo "Done"
