module clewcat.com/channeld

go 1.16

require (
	github.com/stretchr/testify v1.7.0
	google.golang.org/protobuf v1.27.1
	rsc.io/getopt v0.0.0-20170811000552-20be20937449
)

replace channeld => ./internal/channeld
