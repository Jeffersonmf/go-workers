.DEFAULT_GOAL := help
.PHONY: help build run test test-race test-cover fmt fmt-check lint check ci \
        docker-build docker-run clean

help: ## 📖 Lista os comandos disponíveis
	@echo "⚙️  go-workers: comandos"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

build: ## 🔨 Compila o binário de exemplo
	@echo "🔨 Compilando..."
	@go build -o bin/go-workers .

run: build ## ▶️  Roda o exemplo (Ctrl+C para parar com shutdown gracioso)
	@echo "▶️  Rodando..."
	@./bin/go-workers

test: ## 🧪 Roda toda a suíte
	@echo "🧪 Rodando testes..."
	@go test ./...

test-race: ## 🧪 Suíte completa com o detector de race (mais lento, mais rigoroso)
	@echo "🧪 Rodando testes com -race..."
	@go test ./... -race

test-cover: ## 🧪 Suíte completa com relatório de cobertura
	@go test ./... -cover

fmt: ## 🎨 Formata o código (gofmt)
	@echo "🎨 Formatando..."
	@gofmt -w .

fmt-check: ## 🎨 Verifica formatação sem alterar nada
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

lint: ## 🔍 golangci-lint (errcheck, govet, staticcheck, gosec, revive, ...)
	@echo "🔍 Rodando golangci-lint..."
	@golangci-lint run ./...

check: fmt-check lint test-race ## ✅ Roda tudo que o CI roda, localmente

ci: check ## 🤖 Alias de check, usado pelo pipeline de CI

docker-build: ## 🐳 Build multi-stage da imagem (binário estático)
	@echo "🐳 Buildando imagem go-workers:latest..."
	@docker build -t go-workers:latest .

docker-run: docker-build ## 🐳 Roda a imagem
	@echo "🚀 Rodando container..."
	@docker run --rm -i go-workers:latest

clean: ## 🧹 Remove artefatos de build
	@echo "🧹 Limpando..."
	@rm -rf bin/
