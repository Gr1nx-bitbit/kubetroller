FROM okteto/golang:1.22

WORKDIR /go/src/

COPY . .

RUN 

CMD ["go", "run", "multi_client.go", "--clusters='prod:./config/config'"]