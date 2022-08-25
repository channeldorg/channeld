package channeld

import (
	"errors"
)

type ChannelAccessType uint8

const (
	ChannelAccessType_Sub    ChannelAccessType = 0
	ChannelAccessType_Unsub  ChannelAccessType = 1
	ChannelAccessType_Remove ChannelAccessType = 2
)

type ChannelAccessLevel uint8

const (
	ChannelAccessLevel_None                ChannelAccessLevel = 0
	ChannelAccessLevel_OwnerOnly           ChannelAccessLevel = 1
	ChannelAccessLevel_OwnerAndGlobalOwner ChannelAccessLevel = 2
	ChannelAccessLevel_Any                 ChannelAccessLevel = 3
)

func (ch *Channel) CheckACL(c ConnectionInChannel, accessType ChannelAccessType) (bool, error) {
	// default acl is none
	acl := ChannelAccessLevel_None

	// get acl from global setting
	channelSettings, exists := GlobalSettings.ChannelSettings[ch.channelType]
	if exists {
		aclSettings := channelSettings.ACLSettings
		switch accessType {
		case ChannelAccessType_Sub:
			acl = aclSettings.Sub
			break
		case ChannelAccessType_Unsub:
			acl = aclSettings.Unsub
			break
		case ChannelAccessType_Remove:
			acl = aclSettings.Remove
			break
		}
	}

	switch acl {
	case ChannelAccessLevel_None:
		return false, errors.New("none can access")

	case ChannelAccessLevel_OwnerOnly:
		if ch.ownerConnection == c {
			return true, nil
		} else {
			return false, errors.New("only the channel owenr can access")
		}
	case ChannelAccessLevel_OwnerAndGlobalOwner:
		if ch.ownerConnection == c || globalChannel.ownerConnection == c {
			return true, nil
		} else {
			return false, errors.New("only the channel owenr or global channel owner can access")
		}
	case ChannelAccessLevel_Any:
		return true, nil
	default:
		return false, errors.New("illegel channel access level")
	}

}
