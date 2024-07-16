grpcGenAuth:
	protoc -I api api/protos/auth_service.proto --go_out=./api/gen --go-grpc_out=./api/gen

grpcGenChat:
	protoc -I api api/protos/chat_service.proto --go_out=./api/gen --go-grpc_out=./api/gen

run:
	go run cmd/main.go

PID_FILE = server.pid
SHELL := /bin/zsh

run-test:
	@go run cmd/main.go & echo $$! > ${PID_FILE}
	@sleep 2
	@go test ./tests/...
	@sleep 2
	@if [ -f $(PID_FILE) ]; then kill $$(cat $(PID_FILE)) && rm $(PID_FILE); fi