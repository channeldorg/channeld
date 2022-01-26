package proto

import (
	"errors"

	protobuf "google.golang.org/protobuf/proto"
)

func (dst *TestMergeMessage) Merge(src protobuf.Message, options *ChannelDataMergeOptions) error {
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
