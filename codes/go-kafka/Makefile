# Go Kafka Makefile

.PHONY: help build test clean setup run-producer run-consumer lint docker-up docker-down

# 默认目标
.DEFAULT_GOAL := help

# 变量
APP_NAME := go-kafka
DOCKER_COMPOSE := docker-compose.yml

help: ## 显示帮助信息
	@echo "Go Kafka 项目命令:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \\033[36m%-20s\\033[0m %s\\n", $$1, $$2}'

build: ## 构建项目
	@echo "正在构建项目..."
	go build -v ./...

test: ## 运行测试
	@echo "运行测试..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "测试覆盖率报告: coverage.html"

clean: ## 清理构建文件
	@echo "清理..."
	rm -f coverage.out coverage.html
	go clean -cache

deps: ## 下载依赖
	@echo "下载依赖..."
	go mod download
	go mod tidy

setup: ## 初始化项目（安装依赖）
	make deps
	@echo "项目初始化完成！"

lint: ## 运行代码检查
	@echo "运行 golangci-lint..."
	golangci-lint run ./...
	
fmt: ## 格式化代码
	@echo "格式化代码..."
	go fmt ./...

# Docker 相关命令
docker-up: ## 启动 Kafka 服务
	@echo "启动 Kafka..."
	docker-compose -f $(DOCKER_COMPOSE) up -d
	@echo "等待 Kafka 启动..."
	@sleep 10
	@echo "Kafka UI: http://localhost:8080"

docker-down: ## 停止 Kafka 服务
	@echo "停止 Kafka..."
	docker-compose -f $(DOCKER_COMPOSE) down

docker-logs: ## 查看 Kafka 日志
	docker-compose -f $(DOCKER_COMPOSE) logs -f

docker-clean: ## 清理 Docker 数据
	docker-compose -f $(DOCKER_COMPOSE) down -v
	docker system prune -f

# 运行示例
run-producer: ## 运行生产者示例
	go run examples/producer_example.go

run-consumer: ## 运行消费者示例
	go run examples/consumer_example.go

run-order-setup: ## 初始化订单系统
	go run examples/order_system.go setup

run-order-producer: ## 运行订单生产者
	go run examples/order_system.go producer

run-order-consumer: ## 运行订单消费者
	go run examples/order_system.go consumer

# Topic 管理命令
topics-list: ## 列出所有 Topics
	@docker exec -it kafka kafka-topics --bootstrap-server localhost:9092 --list

topics-create: ## 创建 Topic (参数: name=<name> partitions=<n>)
	@if [ -z "$(name)" ] || [ -z "$(partitions)" ]; then \\
		echo "使用方法: make topics-create name=my-topic partitions=3"; \\
		exit 1; \\
	fi
	@docker exec -it kafka kafka-topics \\
		--bootstrap-server localhost:9092 \\
		--create \\
		--topic $(name) \\
		--partitions $(partitions) \\
		--replication-factor 1

topics-delete: ## 删除 Topic (参数: name=<name>)
	@if [ -z "$(name)" ]; then \\
		echo "使用方法: make topics-delete name=my-topic"; \\
		exit 1; \\
	fi
	@docker exec -it kafka kafka-topics \\
		--bootstrap-server localhost:9092 \\
		--delete \\
		--topic $(name)

topics-describe: ## 查看 Topic 详情 (参数: name=<name>)
	@if [ -z "$(name)" ]; then \\
		echo "使用方法: make topics-describe name=my-topic"; \\
		exit 1; \\
	fi
	@docker exec -it kafka kafka-topics \\
		--bootstrap-server localhost:9092 \\
		--describe \\
		--topic $(name)

# 消费者组命令
groups-list: ## 列出消费者组
	@docker exec -it kafka kafka-consumer-groups --bootstrap-server localhost:9092 --list

groups-describe: ## 查看消费者组详情 (参数: group=<name>)
	@if [ -z "$(group)" ]; then \\
		echo "使用方法: make groups-describe group=my-group"; \\
		exit 1; \\
	fi
	@docker exec -it kafka kafka-consumer-groups \\
		--bootstrap-server localhost:9092 \\
		--describe \\
		--group $(group)

groups-reset: ## 重置消费者组偏移量 (参数: group=<name> topic=<name>)
	@if [ -z "$(group)" ] || [ -z "$(topic)" ]; then \\
		echo "使用方法: make groups-reset group=my-group topic=my-topic"; \\
		exit 1; \\
	fi
	@docker exec -it kafka kafka-consumer-groups \\
		--bootstrap-server localhost:9092 \\
		--group $(group) \\
		--topic $(topic) \\
		--reset-offsets \\
		--to-earliest \\
		--execute

# 性能测试命令
perf-producer: ## 生产者性能测试
	docker exec -it kafka kafka-producer-perf-test \\
		--topic test-perf \\
		--num-records 100000 \\
		--record-size 1024 \\
		--throughput -1 \\
		--producer-props bootstrap.servers=localhost:9092

perf-consumer: ## 消费者性能测试
	docker exec -it kafka kafka-consumer-perf-test \\
		--topic test-perf \\
		--messages 100000 \\
		--bootstrap-server localhost:9092

# 生产环境命令（示例）
prod-build: ## 生产构建
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/app .

prod-docker: ## 构建 Docker 镜像
	docker build -t $(APP_NAME):latest .

# 开发命令
dev: ## 启动开发环境（自动重启）
	@which air > /dev/null || (echo "安装 air..." && go install github.com/cosmtrek/air@latest)
	air