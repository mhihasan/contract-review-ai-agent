.PHONY: migrate-up migrate-down sqlc-generate

migrate-up:
	mise exec -- migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	mise exec -- migrate -path migrations -database "$(DATABASE_URL)" down

sqlc-generate:
	mise exec -- sqlc generate
