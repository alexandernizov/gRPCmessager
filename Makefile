lint:
	golangci-lint run



run:
	go run cmd/messanger/main.go

test:
	go clean -testcache
	go test ./...



build:
	echo "Building messanger-app"
	GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-gcc go build -o messanger ./cmd/messanger

restart:
	docker-compose down
	docker rmi grpcmessager-messanger:latest || true
	docker-compose build
	docker-compose up -d

rebuild: build restart

generate:
	go generate ./...

gen-auth:
	protoc -I ./gen/protos ./gen/protos/auth_service.proto --go_out=./gen/ --go-grpc_out=./gen/

gen-chat:
	protoc -I ./gen/protos ./gen/protos/chat_service.proto --go_out=./gen/ --go-grpc_out=./gen/

gen-outbox:
	protoc -I ./api/protos ./api/protos/outbox.proto --go_out=./api/ --go-grpc_out=.api/



migrate-up:
	migrate -path ./migrations -database 'postgres://postgres:password@localhost:5432/postgres?sslmode=disable' up

migrate-down:
	migrate -path ./migrations -database 'postgres://postgres:password@localhost:5432/postgres?sslmode=disable' down