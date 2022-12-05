package chatpb

import (
	"errors"
	"time"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
)

// Filter the merged list when the length of the merged list exceeds mergeOptions.ListSizeLimit
var TimeSpanLimit time.Duration = time.Second * 10

func (dst *ChatChannelData) Merge(src common.ChannelDataMessage, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) error {
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

	limit := int(options.ListSizeLimit)
	n := len(dst.ChatMessages)
	if limit > 0 && n > limit {
		if options.TruncateTop {
			start := n - limit
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
			dst.ChatMessages = dst.ChatMessages[:limit]
		}
	}

	return nil
}
