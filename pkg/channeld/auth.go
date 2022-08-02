package channeld

import (
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"go.uber.org/zap"
)

type AuthProvider interface {
	DoAuth(pit string, lt string) (channeldpb.AuthResultMessage_AuthResult, error)
}

// Do nothing but logging
type LoggingAuthProvider struct {
	Logger *zap.Logger
	Msg    string
}

func (provider *LoggingAuthProvider) DoAuth(pit string, lt string) (channeldpb.AuthResultMessage_AuthResult, error) {
	if provider.Logger != nil {
		provider.Logger.Info(provider.Msg, zap.String("pit", pit), zap.String("lt", lt))
	}
	return channeldpb.AuthResultMessage_SUCCESSFUL, nil
}

var authProvider AuthProvider //= &LoggingAuthProvider{
// 	Logger: zap.NewExample(),
// 	Msg:    "do auth",
// }

func SetAuthProvider(value AuthProvider) {
	authProvider = value
}
