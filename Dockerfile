FROM alpine:latest

WORKDIR /root/

COPY messanger-app .

COPY ./configs/prod.yaml .

COPY ./migrations/ ./migrations/

RUN apk update
RUN apk add postgresql-client

RUN chmod +x wait-for-postgres.sh

EXPOSE 50001

CMD ["./messanger", "-config", "./prod.yaml"]