.PHONY: backend-run backend-build backend-swagger frontend-run frontend-build all

all: backend-swagger backend-build frontend-build

backend-swagger:
	cd backend && swag init

backend-build:
	cd backend && go build -o ../bin/backend main.go

backend-run: backend-swagger
	cd backend && go run main.go

frontend-install:
	cd frontend && npm install

frontend-run:
	cd frontend && npm start

frontend-build:
	cd frontend && npm run build
