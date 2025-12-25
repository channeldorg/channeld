FROM golang:1.25.5

COPY . /channeld/
#RUN ls -la /channeld
WORKDIR /channeld/examples/chat-rooms

ENV GOPROXY="https://goproxy.io"
RUN go get -d -v ./...
RUN go install -v ./...
RUN go build -o app

EXPOSE 8080
EXPOSE 12108

CMD ["./app"]