run:
	go run cmd/messanger/main.go

test:
	go clean -testcache
	go test ./...

build-docker:
	echo "Building messanger-app"
	GOOS=linux GOARCH=amd64 go build -o  messanger ./cmd/messanger
	echo "Starting docker-compose"
	docker-compose up --build messanger


generate:
	go generate ./...

gen-auth:
	protoc -I ./gen/protos ./gen/protos/auth_service.proto --go_out=./gen/ --go-grpc_out=./gen/

gen-chat:
	protoc -I ./gen/protos ./gen/protos/chat_service.proto --go_out=./gen/ --go-grpc_out=./gen/

gen-oapi:
	oapi-codegen -generate chi-server -o ./api/gen/http/http.go -package http ./api/swagger/http.yaml

migrate-up:
	migrate -path ./migrations -database 'postgres://postgres:password@localhost:5432/postgres?sslmode=disable' up

migrate-down:
	migrate -path ./migrations -database 'postgres://postgres:password@localhost:5432/postgres?sslmode=disable' down
