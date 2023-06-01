module github.com/metaworking/channeld

go 1.18

require (
	github.com/golang/snappy v0.0.4
	github.com/gorilla/websocket v1.4.2
	github.com/indiest/fmutils v0.1.2
	github.com/pkg/profile v1.6.0
	github.com/prometheus/client_golang v1.11.1
	github.com/stretchr/testify v1.8.3
	github.com/xtaci/kcp-go v5.4.20+incompatible
	go.uber.org/zap v1.19.1
	google.golang.org/protobuf v1.30.0
	nhooyr.io/websocket v1.8.7
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gin-gonic/gin v1.9.1 // indirect
	github.com/golang/protobuf v1.5.0 // indirect
	github.com/klauspost/compress v1.10.3 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/klauspost/reedsolomon v1.9.14 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mennanov/fmutils v0.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.26.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/puzpuzpuz/xsync/v2 v2.4.0
	github.com/rogpeppe/go-internal v1.8.0 // indirect
	github.com/templexxx/cpufeat v0.0.0-20180724012125-cef66df7f161 // indirect
	github.com/templexxx/xor v0.0.0-20191217153810-f85b25db303b // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/xtaci/lossyconn v0.0.0-20200209145036-adba10fffc37 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/crypto v0.9.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

//replace google.golang.org/protobuf => ../protobuf-go
