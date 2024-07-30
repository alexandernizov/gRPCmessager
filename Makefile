gen:
	go generate ./...

run:
	go run cmd/messanger/main.go

migrate-up:
	migrate -path ./migrations -database 'postgres://postgres:password@localhost:5432/postgres?sslmode=disable' up

migrate-down:
	migrate -path ./migrations -database 'postgres://postgres:password@localhost:5432/postgres?sslmode=disable' down

docker-run-postgres:
	docker run -p 5432:5432 --name messPostgres -e POSTGRESS_PASSWORD=password -d postgres

build-docker:
	echo "Building auth-app"
	GOOS=linux GOARCH=amd64 go build -o auth-app ./cmd/authapp && \
	echo "Building chat-app"
	GOOS=linux GOARCH=amd64 go build -o chat-app ./cmd/chatapp && \
	echo "Starting docker-compose"
	docker-compose up --build auth-app chat-app