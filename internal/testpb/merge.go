package testpb

import (
	"errors"

	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/channeldorg/channeld/pkg/common"
	"google.golang.org/protobuf/proto"
)

// Implement [channeld.MergeableChannelData]
func (dst *TestMergeMessage) Merge(src proto.Message, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) error {
	srcMsg, ok := src.(*TestMergeMessage)
	if !ok {
		return errors.New("src is not a TestMergeMessage")
	}

	if options.ShouldReplaceList {
		// Make a deep copy
		dst.List = append([]string{}, srcMsg.List...)
	} else {
		dst.List = append(dst.List, srcMsg.List...)
	}

	if options.ListSizeLimit > 0 {
		if options.TruncateTop {
			start := len(dst.List) - int(options.ListSizeLimit)
			if start < 0 {
				start = 0
			}
			dst.List = dst.List[start:]
		} else {
			dst.List = dst.List[:options.ListSizeLimit]
		}
	}

	for k, v := range srcMsg.Kv {
		if v.Removed {
			delete(dst.Kv, k)
		} else {
			dst.Kv[k] = v
		}
	}
	return nil
}
