FROM golang:1.16

COPY . /channeld/
#RUN ls -la /channeld
WORKDIR /channeld/examples/chat-rooms

RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go get -d -v ./...
RUN go install -v ./...
RUN go build -o app

EXPOSE 8080
EXPOSE 12108

CMD ["./app"]