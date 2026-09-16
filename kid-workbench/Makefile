.PHONY: up down logs ps seed refresh-math

up:
	./scripts/docker-up.sh

down:
	./scripts/docker-up.sh --down

logs:
	docker compose logs -f

ps:
	docker compose ps

seed:
	docker compose run --rm seed

seed-demo:
	docker compose run --rm --entrypoint ./seed seed -mode=demo -days=60

refresh-math:
	docker compose rm -sf math-app
	docker compose up --build --no-deps -d math-app
