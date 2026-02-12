.PHONY: backend-run backend-build backend-generate backend-fmt frontend-run frontend-build frontend-fmt fmt all

all: backend-build frontend-build

fmt: backend-fmt frontend-fmt

backend-fmt:
	cd backend && go fmt ./...

backend-generate:
	cd backend && go generate ./...&& cd ../frontend && npm run generate-api

backend-build: backend-generate
	cd backend && go build -o ../bin/backend .

backend-run: backend-generate
	cd backend && go run . --debug

frontend-install:
	cd frontend && npm install

frontend-run:
	cd frontend && npm start

frontend-fmt:
	cd frontend && npm run fmt

frontend-build:
	cd frontend && npm run build