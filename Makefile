run:
	go run cmd/messanger/main.go

test:
	go clean -testcache
	go test ./...

build-docker:
	echo "Building messanger-app"
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-gcc go build -o messanger --ldflags '-linkmode external -extldflags "-static"' -tags musl ./cmd/messanger

generate:
	go generate ./...

gen-auth:
	protoc -I ./gen/protos ./gen/protos/auth_service.proto --go_out=./gen/ --go-grpc_out=./gen/

gen-chat:
	protoc -I ./gen/protos ./gen/protos/chat_service.proto --go_out=./gen/ --go-grpc_out=./gen/


migrate-up:
	migrate -path ./migrations -database 'postgres://postgres:password@localhost:5432/postgres?sslmode=disable' up

migrate-down:
	migrate -path ./migrations -database 'postgres://postgres:password@localhost:5432/postgres?sslmode=disable' down

lint:
	golangci-lint run