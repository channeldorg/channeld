FROM golang:1.16

COPY . /channeld

WORKDIR /channeld

ENV GOPROXY="https://goproxy.io"
RUN go get -d -v ./...
RUN go install -v ./...
RUN go build -o app ./cmd

EXPOSE 8080
EXPOSE 11288

CMD ["./app"]