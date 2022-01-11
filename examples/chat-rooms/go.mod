module channeld.clewcat.com/examples/chat

go 1.16

require (
	channeld.clewcat.com/channeld v0.0.0-00010101000000-000000000000
	github.com/golang/snappy v0.0.4 // indirect
	github.com/pkg/profile v1.6.0 // indirect
	github.com/prometheus/client_golang v1.11.0
)

replace channeld.clewcat.com/channeld => ../../../channeld
