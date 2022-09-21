FROM golang:1.16

COPY . /channeld

WORKDIR /channeld/examples/channeld-ue-tps

ENV GOPROXY="https://goproxy.io"
RUN go get -d -v ./...
RUN go install -v ./...
RUN go build -o app .

EXPOSE 12108
EXPOSE 11288

CMD ["./app"]