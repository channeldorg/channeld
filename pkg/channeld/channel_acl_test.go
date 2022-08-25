package channeld

import (
	"testing"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"github.com/stretchr/testify/assert"
)

type aclTestConnection struct {
	id ConnectionId
}

func (c *aclTestConnection) Id() ConnectionId {
	return 0
}

func (c *aclTestConnection) GetConnectionType() channeldpb.ConnectionType {
	return channeldpb.ConnectionType_NO_CONNECTION
}

func (c *aclTestConnection) OnAuthenticated() {

}

func (c *aclTestConnection) HasAuthorityOver(ch *Channel) bool {
	return false
}

func (c *aclTestConnection) Close() {
}

func (c *aclTestConnection) IsClosing() bool {
	return false
}

func (c *aclTestConnection) Send(ctx MessageContext) {

}

func (c *aclTestConnection) SubscribeToChannel(ch *Channel, options *channeldpb.ChannelSubscriptionOptions) *ChannelSubscription {
	return &ChannelSubscription{
		options: channeldpb.ChannelSubscriptionOptions{},
	}
}

func (c *aclTestConnection) UnsubscribeFromChannel(ch *Channel) (*channeldpb.ChannelSubscriptionOptions, error) {
	return nil, nil
}

func (c *aclTestConnection) sendSubscribed(ctx MessageContext, ch *Channel, connToSub ConnectionInChannel, stubId uint32, subOptions *channeldpb.ChannelSubscriptionOptions) {

}

func (c *aclTestConnection) sendUnsubscribed(ctx MessageContext, ch *Channel, connToUnsub *Connection, stubId uint32) {

}

func (c *aclTestConnection) Logger() *Logger {
	return rootLogger
}

var idCounter = 0

func createACLTestConnectionById(id ConnectionId) *aclTestConnection {
	return &aclTestConnection{
		id: id,
	}
}

func createACLTestConnection() *aclTestConnection {
	idCounter++
	return createACLTestConnectionById(ConnectionId(idCounter))
}

func createChannelForTestACL(t channeldpb.ChannelType, owner ConnectionInChannel) (*Channel, error) {
	if t == channeldpb.ChannelType_GLOBAL {
		ch, err := CreateChannel(channeldpb.ChannelType_SUBWORLD, owner)
		ch.channelType = channeldpb.ChannelType_GLOBAL
		globalChannel = ch
		return ch, err
	} else {
		ch, err := CreateChannel(t, owner)
		return ch, err
	}
}

func setChannelACLSettings(cTypes []channeldpb.ChannelType, acl ChannelAccessLevel) {
	for _, t := range cTypes {
		GlobalSettings.ChannelSettings[t] = ChannelSettingsType{
			ACLSettings: ACLSettingsType{
				Sub:    acl,
				Unsub:  acl,
				Remove: acl,
			},
		}
	}
}

func TestCheckACL(t *testing.T) {
	InitLogs()

	operations := []ChannelAccessType{ChannelAccessType_Sub, ChannelAccessType_Unsub, ChannelAccessType_Remove}

	const ChannelType_Test1 channeldpb.ChannelType = 201
	allChannelTypesForTest := []channeldpb.ChannelType{channeldpb.ChannelType_GLOBAL, channeldpb.ChannelType_SUBWORLD, channeldpb.ChannelType_PRIVATE, ChannelType_Test1}
	allChannelTypesForTestWithoutGlobal := []channeldpb.ChannelType{channeldpb.ChannelType_SUBWORLD, channeldpb.ChannelType_PRIVATE, ChannelType_Test1}

	var channelOwner ConnectionInChannel
	var ch *Channel
	var hasAccess bool
	var err error

	for _, o := range operations {

		setChannelACLSettings(allChannelTypesForTest, ChannelAccessLevel_None)

		// CA01
		func(cTypes []channeldpb.ChannelType) {
			for _, cType := range cTypes {
				ch, _ = createChannelForTestACL(cType, createACLTestConnection())
				hasAccess, err = ch.CheckACL(createACLTestConnection(), o)
				assert.EqualError(t, err, "none can access")
				assert.EqualValues(t, hasAccess, false)
			}
		}(allChannelTypesForTest)

		// CA02
		func(cTypes []channeldpb.ChannelType) {
			for _, cType := range cTypes {
				channelOwner = createACLTestConnection()
				ch, _ = createChannelForTestACL(cType, channelOwner)
				hasAccess, err = ch.CheckACL(channelOwner, o)
				assert.EqualError(t, err, "none can access")
				assert.EqualValues(t, hasAccess, false)
			}
		}(allChannelTypesForTest)

		// CA03
		func(cTypes []channeldpb.ChannelType) {
			// set global channel and owner
			globalOwner := createACLTestConnection()
			createChannelForTestACL(channeldpb.ChannelType_GLOBAL, globalOwner)
			for _, cType := range cTypes {
				ch, _ := createChannelForTestACL(cType, createACLTestConnection())
				hasAccess, err := ch.CheckACL(globalOwner, o)
				assert.EqualError(t, err, "none can access")
				assert.EqualValues(t, hasAccess, false)
			}
		}(allChannelTypesForTestWithoutGlobal)

		setChannelACLSettings(allChannelTypesForTest, ChannelAccessLevel_OwnerOnly)

		// CA04
		func(cTypes []channeldpb.ChannelType) {
			for _, cType := range cTypes {
				ch, _ := createChannelForTestACL(cType, createACLTestConnection())
				hasAccess, err := ch.CheckACL(createACLTestConnection(), o)
				assert.EqualError(t, err, "only the channel owenr can access")
				assert.EqualValues(t, hasAccess, false)
			}
		}(allChannelTypesForTest)

		// CA05
		func(cTypes []channeldpb.ChannelType) {
			for _, cType := range cTypes {
				channelOwner := createACLTestConnection()
				ch, _ := createChannelForTestACL(cType, channelOwner)
				hasAccess, err := ch.CheckACL(channelOwner, o)
				assert.NoError(t, err)
				assert.EqualValues(t, hasAccess, true)
			}
		}(allChannelTypesForTest)

		// CA06
		func(cTypes []channeldpb.ChannelType) {
			// set global channel and owner
			globalOwner := createACLTestConnection()
			createChannelForTestACL(channeldpb.ChannelType_GLOBAL, globalOwner)
			for _, cType := range cTypes {
				ch, _ := createChannelForTestACL(cType, createACLTestConnection())
				hasAccess, err := ch.CheckACL(globalOwner, o)
				assert.EqualError(t, err, "only the channel owenr can access")
				assert.EqualValues(t, hasAccess, false)
			}
		}(allChannelTypesForTestWithoutGlobal)

		// CA07
		func(cTypes []channeldpb.ChannelType) {
			for _, cType := range cTypes {
				ch, _ := createChannelForTestACL(cType, nil)
				hasAccess, err := ch.CheckACL(createACLTestConnection(), o)
				assert.EqualError(t, err, "only the channel owenr can access")
				assert.EqualValues(t, hasAccess, false)
			}
		}(allChannelTypesForTest)

		setChannelACLSettings(allChannelTypesForTest, ChannelAccessLevel_OwnerAndGlobalOwner)

		// CA08
		func(cTypes []channeldpb.ChannelType) {
			for _, cType := range cTypes {
				ch, _ = createChannelForTestACL(cType, createACLTestConnection())
				hasAccess, err = ch.CheckACL(createACLTestConnection(), o)
				assert.EqualError(t, err, "only the channel owenr or global channel owner can access")
				assert.EqualValues(t, hasAccess, false)
			}
		}(allChannelTypesForTest)

		// CA09
		func(cTypes []channeldpb.ChannelType) {
			for _, cType := range cTypes {
				channelOwner := createACLTestConnection()
				ch, _ := createChannelForTestACL(cType, channelOwner)
				hasAccess, err := ch.CheckACL(channelOwner, o)
				assert.NoError(t, err)
				assert.EqualValues(t, hasAccess, true)
			}
		}(allChannelTypesForTest)

		// CA10
		func(cTypes []channeldpb.ChannelType) {
			// set global channel and owner
			globalOwner := createACLTestConnection()
			createChannelForTestACL(channeldpb.ChannelType_GLOBAL, globalOwner)
			for _, cType := range cTypes {
				ch, _ := createChannelForTestACL(cType, createACLTestConnection())
				hasAccess, err := ch.CheckACL(globalOwner, o)
				assert.NoError(t, err)
				assert.EqualValues(t, hasAccess, true)
			}
		}(allChannelTypesForTestWithoutGlobal)

		// CA11
		func(cTypes []channeldpb.ChannelType) {
			// set global channel and owner
			globalOwner := createACLTestConnection()
			createChannelForTestACL(channeldpb.ChannelType_GLOBAL, globalOwner)
			for _, cType := range cTypes {
				ch, _ := createChannelForTestACL(cType, nil)
				hasAccess, err := ch.CheckACL(globalOwner, o)
				assert.NoError(t, err)
				assert.EqualValues(t, hasAccess, true)
			}
		}(allChannelTypesForTestWithoutGlobal)

		setChannelACLSettings(allChannelTypesForTest, ChannelAccessLevel_Any)

		//CA12
		func(cTypes []channeldpb.ChannelType) {
			for _, cType := range cTypes {
				ch, _ = createChannelForTestACL(cType, createACLTestConnection())
				hasAccess, err = ch.CheckACL(createACLTestConnection(), o)
				assert.NoError(t, err)
				assert.EqualValues(t, hasAccess, true)
			}
		}(allChannelTypesForTest)

		// CA13
		func(cTypes []channeldpb.ChannelType) {
			for _, cType := range cTypes {
				channelOwner := createACLTestConnection()
				ch, _ := createChannelForTestACL(cType, channelOwner)
				hasAccess, err := ch.CheckACL(channelOwner, o)
				assert.NoError(t, err)
				assert.EqualValues(t, hasAccess, true)
			}
		}(allChannelTypesForTest)

		// CA14
		func(cTypes []channeldpb.ChannelType) {
			// set global channel and owner
			globalOwner := createACLTestConnection()
			createChannelForTestACL(channeldpb.ChannelType_GLOBAL, globalOwner)
			for _, cType := range cTypes {
				ch, _ := createChannelForTestACL(cType, nil)
				hasAccess, err := ch.CheckACL(globalOwner, o)
				assert.NoError(t, err)
				assert.EqualValues(t, hasAccess, true)
			}
		}(allChannelTypesForTestWithoutGlobal)

		// CA15
		func(cTypes []channeldpb.ChannelType) {
			// set global channel and owner
			globalOwner := createACLTestConnection()
			createChannelForTestACL(channeldpb.ChannelType_GLOBAL, globalOwner)
			for _, cType := range cTypes {
				ch, _ := createChannelForTestACL(cType, nil)
				hasAccess, err := ch.CheckACL(globalOwner, o)
				assert.NoError(t, err)
				assert.EqualValues(t, hasAccess, true)
			}
		}(allChannelTypesForTestWithoutGlobal)

	}

	// ch = nil
	// ch.CheckACL(ownerConn, o)
}
