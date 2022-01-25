module channeld.clewcat.com/channeld

go 1.16

require (
	github.com/golang/snappy v0.0.4
	github.com/gorilla/websocket v1.4.2
	github.com/iancoleman/strcase v0.2.0
	github.com/indiest/fmutils v0.1.2
	github.com/klauspost/reedsolomon v1.9.14 // indirect
	github.com/mennanov/fmutils v0.1.1 // indirect
	github.com/pkg/profile v1.6.0
	github.com/prometheus/client_golang v1.11.0
	github.com/stretchr/testify v1.7.0
	github.com/templexxx/cpufeat v0.0.0-20180724012125-cef66df7f161 // indirect
	github.com/templexxx/xor v0.0.0-20191217153810-f85b25db303b // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/xtaci/kcp-go v5.4.20+incompatible
	github.com/xtaci/lossyconn v0.0.0-20200209145036-adba10fffc37 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.19.1
	google.golang.org/protobuf v1.27.1
	nhooyr.io/websocket v1.8.7
)

//replace google.golang.org/protobuf => ../protobuf-go
