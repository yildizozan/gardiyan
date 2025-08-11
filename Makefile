SHELL := /bin/bash

.PHONY: help build run test clean docker-build docker-run

help: ## Bu yardım mesajını göster
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Binary'yi oluştur
	go build -o gardiyan main.go

run: ## Server'ı çalıştır
	go run main.go

test: ## Testleri çalıştır
	go test -v ./...

test-coverage: ## Test coverage raporu oluştur
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean: ## Build dosyalarını temizle
	rm -f gardiyan coverage.out

deps: ## Go modül bağımlılıklarını indir
	go mod download
	go mod tidy

docker-build: ## Docker image'ı oluştur
	docker build -t gardiyan:latest .

docker-run: ## Docker container'ı çalıştır
	docker run -p 8080:8080 --env-file .env gardiyan:latest

docker-compose-up: ## Docker Compose ile çalıştır
	docker-compose up --build

docker-compose-down: ## Docker Compose'u durdur
	docker-compose down

fmt: ## Go kodunu formatla
	go fmt ./...

vet: ## Go kodunu vet ile kontrol et
	go vet ./...

lint: ## Golangci-lint ile kod analizi
	golangci-lint run

dev: ## Development ortamı için server'ı restart ile çalıştır
	air

install-dev-tools: ## Development araçlarını yükle
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
