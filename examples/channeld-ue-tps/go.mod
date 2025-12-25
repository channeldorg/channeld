module github.com/channeldorg/channeld/examples/channeld-ue-tps

go 1.25

require (
	github.com/channeldorg/channeld v0.0.0
	github.com/prometheus/client_golang v1.11.1
	github.com/stretchr/testify v1.8.1
	go.uber.org/zap v1.19.1
	google.golang.org/protobuf v1.28.1
)

replace github.com/channeldorg/channeld => ../..
