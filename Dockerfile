FROM okteto/golang:1.22

WORKDIR /go/src/

COPY . .

CMD ["go", "run", "multi_client.go", "--clusters='prod:./config/config'"]