package channeld

import (
	"github.com/channeldorg/channeld/pkg/channeldpb"
	"go.uber.org/zap"
)

type AuthProvider interface {
	DoAuth(connId ConnectionId, pit string, lt string) (channeldpb.AuthResultMessage_AuthResult, error)
}

// Do nothing but logging
type LoggingAuthProvider struct {
	Logger *zap.Logger
	Msg    string
}

func (provider *LoggingAuthProvider) DoAuth(connId ConnectionId, pit string, lt string) (channeldpb.AuthResultMessage_AuthResult, error) {
	if provider.Logger != nil {
		provider.Logger.Info(provider.Msg, zap.Uint32("connId", uint32(connId)), zap.String("pit", pit), zap.String("lt", lt))
	}
	return channeldpb.AuthResultMessage_SUCCESSFUL, nil
}

// Always return AuthResultMessage_INVALID_LT
type AlwaysFailAuthProvider struct{}

func (provider *AlwaysFailAuthProvider) DoAuth(connId ConnectionId, pit string, lt string) (channeldpb.AuthResultMessage_AuthResult, error) {
	return channeldpb.AuthResultMessage_INVALID_PIT, nil
}

type FixedPasswordAuthProvider struct {
	Password string
}

func (provider *FixedPasswordAuthProvider) DoAuth(connId ConnectionId, pit string, lt string) (channeldpb.AuthResultMessage_AuthResult, error) {
	if lt == provider.Password {
		return channeldpb.AuthResultMessage_SUCCESSFUL, nil
	} else {
		return channeldpb.AuthResultMessage_INVALID_LT, nil
	}
}

var authProvider AuthProvider

// = &AlwaysFailAuthProvider{}

//= &LoggingAuthProvider{
// 	Logger: zap.NewExample(),
// 	Msg:    "do auth",
// }

func SetAuthProvider(value AuthProvider) {
	authProvider = value
}
