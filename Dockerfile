FROM golang:1.16

COPY . /kooola-channeld/
#RUN ls -la /channeld
WORKDIR /kooola-channeld/

ENV GOPROXY="https://goproxy.io"
RUN go get -d -v .
RUN go install -v .
RUN go build -o app

EXPOSE 11288
EXPOSE 12108

CMD ["./app", "-chs=./config/channel_settings_lofi.json", "-loglevel=0"]