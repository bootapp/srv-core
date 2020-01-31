package server

import (
	"context"
	"encoding/json"
	"github.com/bootapp/rest-grpc-oauth2/auth"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/shima-park/agollo"
	"os"
	"srv-core/oauth"
	"srv-core/settings"
)
type ConfigYml struct {
	oauth struct {
		server struct {
			clientList string
			privateKey string
		}
}
}
func ApolloConfig(ctx context.Context, cacheOnly bool, server *oauth.UserPassOAuthServer, authenticator *auth.StatelessAuthenticator) {
	err := agollo.InitWithDefaultConfigFile(
		agollo.WithLogger(agollo.NewLogger(agollo.LoggerWriter(os.Stdout))),
		agollo.AutoFetchOnCacheMiss(),
		agollo.FailTolerantOnBackupExists(),
	)
	if err != nil {
		panic(err)
	}
	//=========== init oauth2 server clients
	var configStr string
	var clients map[string] string
	configStr = agollo.Get("oauth.server.clientList", agollo.WithNamespace("oauth-server"))
	err = json.Unmarshal([]byte(configStr), &clients)
	if err != nil {
		panic(err)
	}
	server.UpdateClientStore(clients)
	//=========== init oauth2 server private key
	configStr = agollo.Get("oauth.server.privateKey", agollo.WithNamespace("oauth-server"))
	server.SetRSAKeyFromPem([]byte(configStr))
	//=========== init user authenticator
	configStr = agollo.Get("oauth.tokenKey")
	authenticator.UpdateKey([]byte(configStr), jwt.SigningMethodRS256)
	configStr = agollo.Get("oauth.clientId")
	secretStr := agollo.Get("oauth.clientSecret")
	serverStr := agollo.Get("oauth.serverAddr")
	authenticator.SetOauthClient(serverStr, configStr, secretStr)
	configStr = agollo.Get("grpcServiceAddr.dal.core")
	settings.DalCoreUserAddr = configStr
	settings.DalCoreSysAddr = configStr
	authenticator.SetAuthorityEndpoint(configStr)
	server.SetupUserClient(configStr)
	err = authenticator.ScheduledFetchAuthorities(ctx)
	if err != nil {
		glog.Fatal(err)
	}
	if !cacheOnly {
		return
	}
	errorCh := agollo.Start()
	watchCh := agollo.Watch()
	//============ listen to user changes
	go func() {
		for {
			select {
			case err := <-errorCh:
				glog.Error(err)
			case update := <-watchCh:
				for k,v := range update.NewValue {
					if update.OldValue[k] != v {
						switch k {
						case "oauth.server.clientList":
							err = json.Unmarshal([]byte(v.(string)), &clients)
							if err != nil {
								glog.Error(err)
								break
							}
							server.UpdateClientStore(clients)
							glog.Info("oauth-server client list updated.")
						case "oauth.server.privateKey":
							server.SetRSAKeyFromPem([]byte(v.(string)))
							glog.Info("oauth-server keys updated.")
						case "oauth.tokenKey":
							authenticator.UpdateKey([]byte(configStr), jwt.SigningMethodRS256)
							glog.Info("application tokenKey updated.")
						case "oauth.clientId":
							authenticator.SetOauthClientId(v.(string))
							glog.Info("application clientId updated.")
						case "oauth.clientSecret":
							authenticator.SetOauthClientSecret(v.(string))
							glog.Info("application clientSecret updated.")


						}

					}
				}
			}
		}
	}()
}