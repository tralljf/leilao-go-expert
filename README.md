# Sistema de leiloes com fechamento automatico

Projeto do desafio Go Expert (Full Cycle) para adicionar fechamento automatico de leiloes usando goroutines, sobre o repositorio base [labs-auction-goexpert](https://github.com/devfullcycle/labs-auction-goexpert).

## Como funciona

Ao criar um leilao (`internal/infra/database/auction/create_auction.go`), o repositorio grava o documento no MongoDB e dispara uma goroutine que aguarda ate `timestamp do leilao + AUCTION_DURATION`. Quando o tempo expira, ela faz um `UpdateOne` no MongoDB alterando o status de `Active` (0) para `Completed` (1, o status "fechado" do projeto base). A criacao do leilao nao fica bloqueada: a requisicao retorna assim que o documento e inserido.

O filtro do update inclui `status: Active`, entao um leilao que ja esteja fechado nao e alterado de novo.

## Como rodar com Docker

```bash
docker compose up -d --build
```

A API fica disponivel em `http://localhost:8080`. Para parar:

```bash
docker compose down
```

### Exemplo de uso

```bash
# Cria um leilao (retorna 201)
curl -X POST localhost:8080/auction \
  -H 'Content-Type: application/json' \
  -d '{"product_name":"Notebook","category":"Eletronicos","description":"Notebook usado em bom estado","condition":1}'

# Lista leiloes abertos (status=0) para obter o id
curl 'localhost:8080/auction?status=0'

# Depois de AUCTION_DURATION, o status do leilao passa a ser 1 (Completed)
curl localhost:8080/auction/<id>
```

## Variaveis de ambiente

As variaveis ficam em `cmd/auction/.env` (lidas pela aplicacao e pelo Docker Compose).

```env
BATCH_INSERT_INTERVAL=20s
MAX_BATCH_SIZE=4
AUCTION_INTERVAL=20s
AUCTION_DURATION=20s

MONGO_INITDB_ROOT_USERNAME=admin
MONGO_INITDB_ROOT_PASSWORD=admin
MONGODB_URL=mongodb://admin:admin@mongodb:27017/auctions?authSource=admin
MONGODB_DB=auctions
```

Variaveis de tempo:

- `AUCTION_DURATION`: por quanto tempo um leilao permanece aberto antes de ser fechado automaticamente. Aceita o formato de `time.ParseDuration`, por exemplo `30s`, `5m` ou `1h`.
- `AUCTION_INTERVAL`: variavel original do projeto base. E usada como fallback quando `AUCTION_DURATION` nao esta definida. A validacao de lances (`create_bid.go`) usa a mesma regra, para que lances e fechamento concordem sobre o fim do leilao.
- Se nenhuma das duas for valida, a duracao padrao e `5m`.
- `BATCH_INSERT_INTERVAL` e `MAX_BATCH_SIZE`: controlam o envio em lote dos lances (projeto base).

Para mudar a duracao, edite `AUCTION_DURATION` em `cmd/auction/.env` e recrie o container com `docker compose up -d --build`.

As credenciais do MongoDB acima sao apenas para o ambiente local.

## Como testar

O teste de fechamento automatico precisa de um MongoDB. Suba apenas o MongoDB do Docker Compose (porta 27017 exposta no host):

```bash
docker compose up -d mongodb
go test -race -v ./...
docker compose down
```

O teste `TestCreateAuctionAutomaticallyClosesAuctionAfterConfiguredDuration` (`internal/infra/database/auction/create_auction_test.go`):

1. define `AUCTION_DURATION=200ms`;
2. cria um leilao e confirma que ele nasce com status `Active`;
3. aguarda e verifica que o status mudou para `Completed` sem intervencao manual.

Por padrao o teste conecta em `mongodb://admin:admin@localhost:27017/auctions?authSource=admin` e usa o banco `auction_auto_close_test`. Para usar outro MongoDB, defina `AUCTION_TEST_MONGODB_URL`.
