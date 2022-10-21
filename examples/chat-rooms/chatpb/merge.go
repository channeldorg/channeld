package chatpb

import (
	"errors"
	"time"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"google.golang.org/protobuf/proto"
)

// Filter the merged list when the length of the merged list exceeds mergeOptions.ListSizeLimit
var TimeSpanLimit time.Duration = time.Second * 10

func (dst *ChatChannelData) Merge(src proto.Message, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) error {
	srcMsg, ok := src.(*ChatChannelData)
	if !ok {
		return errors.New("src is not a ChatChannelData")
	}

	if options.ShouldReplaceList {
		// Make a deep copy
		dst.ChatMessages = append([]*ChatMessage{}, srcMsg.ChatMessages...)
	} else {
		dst.ChatMessages = append(dst.ChatMessages, srcMsg.ChatMessages...)
	}

	lsl := int(options.ListSizeLimit)
	n := len(dst.ChatMessages)
	if lsl > 0 && n > lsl {
		if options.TruncateTop {
			start := n - lsl
			if TimeSpanLimit > 0 {
				availableTime := time.Now().Add(-TimeSpanLimit)
				for ; start > 0; start-- {
					if time.UnixMilli(dst.ChatMessages[start-1].SendTime).Before(availableTime) {
						break
					}
				}
			} else if start < 0 {
				start = 0
			}
			dst.ChatMessages = dst.ChatMessages[start:]
		} else {
			dst.ChatMessages = dst.ChatMessages[:lsl]
		}
	}

	return nil
}
