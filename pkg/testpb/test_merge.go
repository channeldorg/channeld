package testpb

import (
	"errors"

	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"google.golang.org/protobuf/proto"
)

// Implement [channeld.MergeableChannelData]
func (dst *TestMergeMessage) Merge(src proto.Message, options *channeldpb.ChannelDataMergeOptions, spatialNotifier channeld.SpatialInfoChangedNotifier) error {
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
