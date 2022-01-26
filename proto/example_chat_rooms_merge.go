package proto

import (
	"errors"

	protobuf "google.golang.org/protobuf/proto"
)

func (dst *ChatChannelData) Merge(src protobuf.Message, options *ChannelDataMergeOptions) error {
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

	if options.ListSizeLimit > 0 {
		if options.TruncateTop {
			start := len(dst.ChatMessages) - int(options.ListSizeLimit)
			if start < 0 {
				start = 0
			}
			dst.ChatMessages = dst.ChatMessages[start:]
		} else {
			dst.ChatMessages = dst.ChatMessages[:options.ListSizeLimit]
		}
	}

	return nil
}
