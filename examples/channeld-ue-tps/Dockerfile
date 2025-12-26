FROM golang:1.25.5-alpine as builder

COPY . /channeld

WORKDIR /channeld/examples/channeld-ue-tps

ENV GOPROXY="https://goproxy.io"
RUN go get -d -v ./...
RUN go install -v ./...
#RUN go build -o app .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o . ./...

FROM alpine as prod

COPY --from=builder /channeld/examples/channeld-ue-tps/channeld-ue-tps .
COPY ./config ./config

EXPOSE 12108
EXPOSE 11288

ENTRYPOINT ["./channeld-ue-tps"]