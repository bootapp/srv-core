package oauth

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"github.com/bootapp/oauth2/errors"
	"github.com/bootapp/oauth2/generates"
	"github.com/bootapp/oauth2/manage"
	"github.com/bootapp/oauth2/models"
	"github.com/bootapp/oauth2/server"
	"github.com/bootapp/oauth2/store"
	core "srv-core/proto"
	"srv-core/settings"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	_ "golang.org/x/crypto/sha3"
	"google.golang.org/grpc"
	"log"
	"time"
)

type UserPassOAuthServer struct {
	privKey *rsa.PrivateKey
	pubKey *rsa.PublicKey
	Srv *server.Server
	clientStore *store.ClientStore
	manager *manage.StatelessManager
	Hash crypto.Hash
	aesKey []byte
	aesCipher cipher.Block
}
var (
	dalCoreUserClient core.DalUserServiceClient
	dalCoreUserConn *grpc.ClientConn
	oauthServer *UserPassOAuthServer
)
func NewPassOAuthServer() *UserPassOAuthServer {
	if oauthServer != nil {
		return oauthServer
	}
	oauthServer = &UserPassOAuthServer{}
	oauthServer.Init()
	return oauthServer
}
func (s *UserPassOAuthServer) Init() {
	s.manager = manage.NewStatelessManager()
	s.clientStore = store.NewClientStore()
	s.manager.MapClientStorage(s.clientStore)
	// Authorize Code Expire Time
	s.manager.SetAuthorizeCodeExp(time.Minute * 10)
	// Password Type Settings
	cfg := &manage.Config {
		// access token expiration time
		AccessTokenExp: time.Hour * 2,
		// refresh token expiration time
		RefreshTokenExp: time.Hour * 24 * 7,
		// whether to generate the refreshing token
		IsGenerateRefresh: true,
	}
	cfgRefresh := &manage.RefreshingConfig{
		RefreshTokenExp: time.Hour * 24 * 7,
		IsGenerateRefresh: true,
		AccessTokenExp: time.Hour * 2,
	}
	s.manager.SetRefreshTokenCfg(cfgRefresh)
	s.manager.SetPasswordTokenCfg(cfg)
	s.Srv = server.NewDefaultServer(s.manager)
	s.Srv.SetAllowGetAccessRequest(true)
	s.Srv.SetClientInfoHandler(server.ClientFormHandler)
	s.Srv.SupportedScope = "user_rw"
	s.Srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
		glog.Error("Internal Error:", err.Error())
		return
	})
	s.Srv.SetResponseErrorHandler(func(re *errors.Response) {
		glog.Error("Response Error:", re.Error.Error())
	})

	s.Srv.SetPasswordAuthorizationHandler(loginHandler)

	s.Hash = crypto.SHA3_256

	s.aesKey = []byte(settings.SignerAESKey)

	var err error
	s.aesCipher, err = aes.NewCipher(s.aesKey)
	if err != nil {
		panic(err)
	}

}

func GetOauthServer() *UserPassOAuthServer {
	return oauthServer
}
func (s * UserPassOAuthServer) SetupUserClient(dalCoreUserAddr string) {
	var err error
	dalCoreUserConn, err = grpc.Dial(dalCoreUserAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	dalCoreUserClient = core.NewDalUserServiceClient(dalCoreUserConn)
}

func (s * UserPassOAuthServer) UpdateClientStore(clients map[string]string) {
	var err error
	for key, value := range clients{
		err = s.clientStore.Set(key, &models.Client {
			ID:     key,
			Secret: value,
			Domain: "http://localhost",
		})
		if err != nil {
			panic(err)
		}
	}
}

func (s *UserPassOAuthServer) SetRSAKeyFromPem(pem []byte) {
	var err error
	s.privKey, err = jwt.ParseRSAPrivateKeyFromPEM(pem)
	if err != nil {
		panic(err)
	}
	s.pubKey = &s.privKey.PublicKey
	s.manager.MapAccessGenerate(generates.NewJWTAccessGenerate(pem, jwt.SigningMethodRS256))
}
func (s *UserPassOAuthServer) GetPublicKey() []byte {
	pubKey, err:= EncodePublicKey(s.pubKey)
	if err != nil {
		glog.Fatal(err)
	}
	return pubKey
}
