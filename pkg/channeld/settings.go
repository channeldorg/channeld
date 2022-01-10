package channeld

import "channeld.clewcat.com/channeld/proto"

type GlobalSettingsType struct {
	CompressionType proto.CompressionType
}

var GlobalSettings = GlobalSettingsType{
	CompressionType: proto.CompressionType_NO_COMPRESSION,
}
