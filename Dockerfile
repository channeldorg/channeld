FROM golang:1.16

COPY . /channeld/
WORKDIR /channeld/cmd

RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go get -d -v ./...
RUN go install -v ./...
RUN go build -o app

EXPOSE 8080
EXPOSE 11288

CMD ["./app"]