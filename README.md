# Correios API

API REST para rastreamento de encomendas por CPF utilizando web scraping com browser automation.

## Funcionalidades

- Rastreamento de encomendas por CPF
- Extração automática de código de rastreio, data prevista e eventos
- Autenticação via API Key
- Rate limiting integrado
- Documentação Swagger

## Tech Stack

| Camada | Tecnologia |
|--------|------------|
| Backend | Go 1.21+, Gin |
| Browser Automation | Rod (headless Chrome) |
| Documentação | Swagger/OpenAPI |
| Container | Docker, Docker Compose |

## Instalação

### Docker (Recomendado)

```bash
docker pull crangelp/correios_api:latest
docker run -p 8087:8087 crangelp/correios_api:latest
```

### Docker Compose

```bash
git clone https://github.com/CRangelP/correios_api.git
cd correios_api
docker-compose up -d
```

### Docker Swarm

```bash
docker stack deploy -c docker-stack.yml correios
```

### Local

```bash
git clone https://github.com/CRangelP/correios_api.git
cd correios_api/backend
cp .env_example .env
# Edite o arquivo .env com suas configurações
go mod tidy
go run ./cmd/api
```

## Configuração

### Variáveis de Ambiente

| Variável | Descrição | Default |
|----------|-----------|---------|
| `PORT` | Porta do servidor | `8087` |
| `API_KEYS` | Chaves de API (separadas por vírgula) | `dev-key-123` |
| `BROWSER_URL` | URL do Chrome remoto (opcional) | - |

### Arquivo .env

Copie o arquivo `.env_example` para `.env` e configure:

```bash
cp backend/.env_example backend/.env
```

```env
# Server Configuration
PORT=8087

# API Keys (comma-separated for multiple keys)
API_KEYS=sua-chave-secreta-aqui

# Browser Configuration (optional - for remote Chrome)
BROWSER_URL=
```

## Endpoints

### Health Check

```http
GET /health
```

**Resposta:**
```json
{"status": "ok"}
```

### Rastrear por CPF

```http
POST /api/v1/tracker/cpf
Content-Type: application/json
X-API-Key: dev-key-123

{
  "cpf": "12345678900"
}
```

**Resposta:**
```json
{
  "success": true,
  "data": {
    "cpf": "12345678900",
    "tracking_code": "AB123456789BR - SEDEX",
    "expected_date": "15/12/2025",
    "status": "em trânsito",
    "events": [
      {
        "date": "12/12/2025 10:30:00",
        "location": "SAO PAULO,SP",
        "location_type": "Unidade de Tratamento",
        "description": "Objeto em transferência - por favor aguarde"
      }
    ],
    "scraped_at": "2025-12-12T18:00:00-03:00"
  },
  "scraping_method": "browser_automation"
}
```

## Status Possíveis

| Status | Descrição |
|--------|-----------|
| `entregue` | Objeto entregue ao destinatário |
| `tentativa de entrega` | Tentativa de entrega não sucedida |
| `saiu para entrega` | Objeto em rota de entrega |
| `aguardando retirada` | Objeto aguardando retirada na agência |
| `em trânsito` | Objeto em transferência entre unidades |
| `postado` | Objeto postado |
| `devolvido` | Objeto devolvido ao remetente |
| `retido na fiscalização` | Objeto retido para fiscalização |
| `extraviado` | Objeto extraviado ou roubado |
| `avariado` | Objeto avariado |
| `aguardando pagamento` | Aguardando pagamento de taxas |
| `etiqueta emitida` | Etiqueta de envio emitida |

## Eventos Suportados

A API captura os seguintes tipos de eventos:

- Objeto em transferência
- Objeto postado
- Objeto entregue
- Objeto não entregue
- Objeto saiu para entrega
- Objeto aguardando retirada
- Objeto devolvido
- Objeto encaminhado
- Objeto retido
- Objeto roubado/extraviado
- Objeto avariado
- Fiscalização aduaneira
- Aguardando/Pagamento confirmado
- Tentativa de entrega
- Objeto coletado
- Coleta solicitada
- Logística reversa
- Destinatário ausente
- Endereço incorreto/insuficiente
- Etiqueta emitida

## Estrutura do Projeto

```
correios_api/
├── backend/
│   ├── cmd/api/
│   │   └── main.go
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handlers/
│   │   │   ├── middleware/
│   │   │   └── routes/
│   │   ├── auth/
│   │   ├── config/
│   │   └── domain/
│   │       └── scraper/
│   ├── docs/
│   ├── .env_example
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
├── api/
│   └── insomnia.json
├── docker-compose.yml
├── docker-stack.yml
├── .gitignore
└── README.md
```

## Testando com Insomnia

Importe o arquivo `api/insomnia.json` no Insomnia para testar todos os endpoints.

## Links

- **GitHub**: https://github.com/CRangelP/correios_api
- **DockerHub**: https://hub.docker.com/r/crangelp/correios_api

## Licença

MIT
