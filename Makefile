run:
	docker-compose up --build

migrate:
	go run ./cmd migrate

seed:
	go run ./cmd seed

build:
	docker-compose build
