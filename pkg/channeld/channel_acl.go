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

var (
	ErrNoneAccess                = errors.New("none can access")
	ErrOwnerOnlyAccess           = errors.New("only the channel owenr can access")
	ErrOwnerAndGlobalOwnerAccess = errors.New("only the channel owenr or global channel owner can access")
	ErrIllegalAccessLevel        = errors.New("illegal channel access level")
)

func (ch *Channel) CheckACL(c ConnectionInChannel, accessType ChannelAccessType) (bool, error) {
	// default level is none
	level := ChannelAccessLevel_None

	// get acl from global setting
	channelSettings, exists := GlobalSettings.ChannelSettings[ch.channelType]
	if exists {
		aclSettings := channelSettings.ACLSettings
		switch accessType {
		case ChannelAccessType_Sub:
			level = aclSettings.Sub
		case ChannelAccessType_Unsub:
			level = aclSettings.Unsub
		case ChannelAccessType_Remove:
			level = aclSettings.Remove
		}
	}

	switch level {
	case ChannelAccessLevel_None:
		return false, ErrNoneAccess

	case ChannelAccessLevel_OwnerOnly:
		if ch.ownerConnection == c {
			return true, nil
		} else {
			return false, ErrOwnerOnlyAccess
		}
	case ChannelAccessLevel_OwnerAndGlobalOwner:
		if ch.ownerConnection == c || globalChannel.ownerConnection == c {
			return true, nil
		} else {
			return false, ErrOwnerAndGlobalOwnerAccess
		}
	case ChannelAccessLevel_Any:
		return true, nil
	default:
		return false, ErrIllegalAccessLevel
	}

}
