# Clima por CEP com Observabilidade

Visão Geral
-----------
Este repositório implementa dois microserviços em Go para resolver temperatura a partir de um CEP:

- `service-a`: recebe requisições do cliente e encaminha para `service-b`.
- `service-b`: resolve cidade com a API da ViaCEP e consulta a temperatura na API WeatherAPI.

Fluxo de comunicação
--------------------
1. Cliente POST `/weather` em `service-a` com JSON `{ "cep": "01001000" }`.
2. `service-a` cria span `input-weather`, injeta o contexto de trace (W3C `traceparent`) e encaminha para `service-b`.
3. `service-b` extrai o contexto, continua o trace com span `weather` e cria spans filhos `lookup_city` e `lookup_temperature`.
4. `service-b` retorna os dados (cidade e temperaturas) para `service-a`, que proxya ao cliente.

Observabilidade (OpenTelemetry + Zipkin)
--------------------------------------
- Os serviços inicializam um `TracerProvider` e exportam traces via OTLP ao Collector (`otel-collector:4317`).
- O Collector está configurado para exportar para Zipkin (UI em `http://localhost:9411`).
- O propagador global está configurado para W3C `TraceContext` e `Baggage` para propagar o mesmo `traceId` entre serviços.
- Os spans incluem atributos e eventos para ajudar a entender payloads e headers transitados.

Execução local com Docker Compose
--------------------------------
Pré-requisitos: Docker e Docker Compose.

Exemplo (PowerShell)

```powershell
# Opcional: apontar para WireMock local (mappings em `test/wiremock`) para testes
$env:VIACEP_BASE_URL='http://wiremock:8080'
$env:WEATHERAPI_BASE_URL='http://wiremock:8080'

# Informe sua chave 
$env:WEATHERAPI_KEY='SUA_WEATHERAPI_KEY_AQUI'

docker compose up --build -d
```

Serviços e portas:

- `service-a`: http://localhost:8080 (POST `/weather`)
- `service-b`: http://localhost:8081
- `zipkin`: http://localhost:9411
- `wiremock` (mappings de teste): http://localhost:8082

Endpoints úteis
---------------
- `POST http://localhost:8080/weather` - body JSON `{ "cep": "01001000" }`
- `GET http://localhost:8080/healthz` - health de `service-a`
- `GET http://localhost:8081/healthz` - health de `service-b`

Variáveis de ambiente importantes
--------------------------------
- `WEATHERAPI_KEY` (obrigatória para chamadas à WeatherAPI em ambiente real). Nunca a coloque em código ou arquivos rastreados por VCS.
- `VIACEP_BASE_URL` e `WEATHERAPI_BASE_URL` - permitem apontar os serviços para endpoints mockados (WireMock) durante testes.

Testes
------
1) Unit tests

 - `service-b` contém um teste unitário que valida o comportamento quando `WEATHERAPI_KEY` não está presente. Para rodar:

```powershell
go test ./service-b
```

2) BDD / testes de integração

 - Os testes BDD baseados em Godog ficam em `test/`. Eles usam WireMock (mappings em `test/wiremock/mappings`) para simular ViaCEP e WeatherAPI.
 - Para rodar os testes BDD localmente:

```powershell
cd test
go test -v
```

Rodando os testes E2E com um script
----------------------------------
Há um script que automatiza o fluxo de levantar o ambiente, executar os testes BDD e aguardar sua confirmação para derrubar os containers:

 - `test/run-e2e.sh` — script shell que:
	1. solicita (se não estiver definida) a variável `WEATHERAPI_KEY` (entrada oculta) e exporta as variáveis necessárias;
	2. sobe o `docker compose up --build -d` (WireMock, services, Collector, Zipkin);
	3. aguarda os endpoints `/healthz` de `service-a` e `service-b` ficarem disponíveis;
	4. executa os testes BDD (`go test -v` dentro de `test/`);
	5. aguarda a digitação de qualquer tecla para derrubar o ambiente (`docker compose down --volumes`).

Como executar o script (exemplo em Linux/macOS/WSL):

```bash
chmod +x test/run-e2e.sh
./test/run-e2e.sh
```

Observações:
- O script pede a `WEATHERAPI_KEY` se ela não estiver definida no ambiente; não armazena a chave em disco.
- Ao final, após revisar os resultados do `go test`, pressione qualquer tecla para derrubar os containers criados.



