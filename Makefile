BACKEND_DIR   := backend
FRONTEND_DIR  := frontend
CLIENT_GO_DIR := client-go
BIN_DIR       := bin

.PHONY: all fmt clean help backend-run backend-build backend-gen frontend-run frontend-build frontend-gen frontend-install client-go-gen

all: frontend-build backend-build client-go-gen ## 构建并编译前后端及 Go 客户端

# --- 后端 (Go) ---
backend-gen: ## 生成 Swagger 文档
	@echo "Updating backend swagger docs..."
	@cd $(BACKEND_DIR) && go generate ./...

backend-build: backend-gen ## 编译后端程序
	@mkdir -p $(BIN_DIR)
	cd $(BACKEND_DIR) && go build -o ../$(BIN_DIR)/backend .

backend-run: backend-gen ## 调试模式运行后端
	cd $(BACKEND_DIR) && go run . --debug

# --- 前端 (Angular) ---
$(FRONTEND_DIR)/node_modules: $(FRONTEND_DIR)/package.json
	@echo "Installing frontend dependencies..."
	cd $(FRONTEND_DIR) && npm install
	@touch $(FRONTEND_DIR)/node_modules

frontend-install: $(FRONTEND_DIR)/node_modules ## 手动触发安装前端依赖

frontend-gen: backend-gen $(FRONTEND_DIR)/node_modules ## 根据 Swagger 文档生成前端 API 客户端
	cd $(FRONTEND_DIR) && npm run generate-api

frontend-build: frontend-gen ## 编译前端页面
	cd $(FRONTEND_DIR) && npm run build

frontend-run: $(FRONTEND_DIR)/node_modules ## 运行前端开发服务器
	cd $(FRONTEND_DIR) && npm start

# --- Go 客户端 (修复路径警告) ---
client-go-gen: $(FRONTEND_DIR)/node_modules ## 生成 Go 客户端代码
	@if ! command -v oapi-codegen > /dev/null; then \
		echo "oapi-codegen not found, installing..."; \
		go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest; \
	fi
	@echo "Step 1: Converting Swagger to OpenAPI 3.0..."
	@mkdir -p $(BACKEND_DIR)/docs
	@npm exec --prefix $(FRONTEND_DIR) -- openapi-generator-cli generate \
		-i $(BACKEND_DIR)/docs/swagger.json \
		-g openapi-yaml \
		-o temp-oas3 > /dev/null 2>&1
	@mv temp-oas3/openapi/openapi.yaml $(BACKEND_DIR)/docs/openapi3.yaml
	@rm -rf temp-oas3

	@echo "Step 2: Preparing client directory..."
	@mkdir -p $(CLIENT_GO_DIR)
	@# 如果子目录没有 go.mod，则初始化
	@if [ ! -f $(CLIENT_GO_DIR)/go.mod ]; then \
		echo "Initializing $(CLIENT_GO_DIR) module..."; \
		cd $(CLIENT_GO_DIR) && go mod init $(CLIENT_GO_DIR); \
	fi

	@echo "Step 3: Generating Go client code (Inside $(CLIENT_GO_DIR))..."
	@# 核心修复：跳转到目录后再执行，并使用相对路径引用 swagger
	@cd $(CLIENT_GO_DIR) && oapi-codegen \
		--package=client \
		--generate="types,client,std-http" \
		-o ./client.gen.go \
		../$(BACKEND_DIR)/docs/openapi3.yaml

	@echo "Step 4: Tidying up dependencies..."
	@cd $(CLIENT_GO_DIR) && go mod tidy

# --- 工具 ---
fmt: backend-gen $(FRONTEND_DIR)/node_modules ## 格式化前后端代码
	cd $(BACKEND_DIR) && go fmt ./...
	cd $(FRONTEND_DIR) && npm run fmt

clean: ## 清理构建产物
	rm -rf $(BIN_DIR)
	rm -rf $(FRONTEND_DIR)/dist
	rm -rf $(FRONTEND_DIR)/node_modules
	rm -f $(CLIENT_GO_DIR)/client.gen.go
	rm -f $(BACKEND_DIR)/docs/openapi3.yaml
	@echo "Cleaned all build artifacts."

help: ## 显示此帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
