BACKEND_DIR  := backend
FRONTEND_DIR := frontend
BIN_DIR      := bin

.PHONY: all fmt clean help backend-run backend-build backend-gen frontend-run frontend-build frontend-gen frontend-install

all: backend-build frontend-build ## 构建并编译前后端

# 后端 (Go)
backend-gen: ## 生成 Swagger 文档
	cd $(BACKEND_DIR) && go generate ./...

backend-build: backend-gen ## 编译后端程序
	mkdir -p $(BIN_DIR)
	cd $(BACKEND_DIR) && go build -o ../$(BIN_DIR)/backend .

backend-run: backend-gen ## 调试模式运行后端 (监听 4200 代理)
	cd $(BACKEND_DIR) && go run . --debug

# 前端 (Angular)
# 自动检测 package.json 变化或 node_modules 是否存在
$(FRONTEND_DIR)/node_modules: $(FRONTEND_DIR)/package.json
	cd $(FRONTEND_DIR) && npm install
	touch $(FRONTEND_DIR)/node_modules

frontend-install: $(FRONTEND_DIR)/node_modules ## 手动触发安装前端依赖

frontend-gen: backend-gen $(FRONTEND_DIR)/node_modules ## 根据 Swagger 文档生成前端 API 客户端
	cd $(FRONTEND_DIR) && npm run generate-api

frontend-build: frontend-gen ## 编译前端页面
	cd $(FRONTEND_DIR) && npm run build

frontend-run: $(FRONTEND_DIR)/node_modules ## 运行前端开发服务器
	cd $(FRONTEND_DIR) && npm start

# 工具
fmt: backend-gen $(FRONTEND_DIR)/node_modules ## 格式化前后端代码
	cd $(BACKEND_DIR) && go fmt ./...
	cd $(FRONTEND_DIR) && npm run fmt

clean: ## 清理构建产物 (bin/ dist/)
	rm -rf $(BIN_DIR) $(FRONTEND_DIR)/dist

help: ## 显示此帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
