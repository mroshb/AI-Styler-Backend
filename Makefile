.PHONY: bot-run bot-test bot-build bot-deploy bot-docker-build bot-docker-run bot-clean

# Bot targets
bot-run:
	@echo "Running Telegram bot..."
	@if [ ! -f .env.bot ]; then \
		echo "❌ .env.bot file not found!"; \
		echo "Creating from example..."; \
		cp configs/example.env .env.bot 2>/dev/null || true; \
		echo "⚠️  Please edit .env.bot and set your TELEGRAM_BOT_TOKEN"; \
		exit 1; \
	fi
	@echo "Loading environment variables from .env.bot..."
	@export $$(cat .env.bot | grep -v '^#' | xargs) && go run cmd/bot/main.go

bot-test:
	@echo "Running bot tests..."
	@go test -v ./tests/telegram/...

bot-build:
	@echo "Building bot binary..."
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/bot cmd/bot/main.go

bot-docker-build:
	@echo "Building bot Docker image..."
	@docker build -f Dockerfile.bot -t ai-styler-bot:latest .

bot-docker-run:
	@echo "Running bot in Docker..."
	@docker-compose -f docker-compose.bot.yml up -d telegram-bot

bot-deploy:
	@echo "Deploying bot..."
	@$(MAKE) bot-docker-build
	@$(MAKE) bot-docker-run

bot-clean:
	@echo "Cleaning bot artifacts..."
	@rm -f bin/bot
	@docker-compose -f docker-compose.bot.yml down

bot-logs:
	@echo "Showing bot logs..."
	@docker-compose -f docker-compose.bot.yml logs -f telegram-bot

bot-stop:
	@echo "Stopping bot..."
	@docker-compose -f docker-compose.bot.yml stop telegram-bot

bot-restart:
	@echo "Restarting bot..."
	@docker-compose -f docker-compose.bot.yml restart telegram-bot

