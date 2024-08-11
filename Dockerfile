FROM alpine:latest

WORKDIR /root/

COPY messanger .

COPY ./configs/prod.yaml .

EXPOSE 50001

#CMD ["./messanger", "-config", "./prod.yaml"]