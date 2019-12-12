package server

import (
	"context"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/bootapp/srv-core/oauth"
	"github.com/bootapp/srv-core/proto/core"
	"github.com/bootapp/srv-core/settings"
	"github.com/bootapp/srv-core/utils"
	"google.golang.org/grpc"
)

type SrvCoreDataStoreServiceServer struct {
	dalCoreUserClient core.DalUserServiceClient
	dalCoreUserConn *grpc.ClientConn
	aliClient *sdk.Client
	oauthServer *oauth.UserPassOAuthServer
}

func NewDataStoreServer() *SrvCoreDataStoreServiceServer {
	s := &SrvCoreDataStoreServiceServer{}
	var err error
	s.aliClient, err = sdk.NewClientWithAccessKey(settings.CredentialSMSRegionId, settings.CredentialSMSAccessKeyId, settings.CredentialSMSAccessSecret)
	if err != nil {
		panic(err)
	}
	s.oauthServer = oauth.GetOauthServer()
	return s
}
func (s *SrvCoreDataStoreServiceServer) QueryUploadToken(ctx context.Context, req *core.UploadTokenReq) (*core.UploadTokenResp, error) {
	var token *core.OSSPolicyToken
	switch req.Type {
	case core.FileDirType_FILE_TYPE_BUSINESS_LICENSE:
		token = utils.GetPolicyToken("business-licenses/")
	}
	return &core.UploadTokenResp{Token:token}, nil
}

func (s *SrvCoreDataStoreServiceServer) InvokeUploadCallback(ctx context.Context, req *core.OSSCallbackReq) (*core.OSSCallbackResp, error) {
	println("callback")
	println(req.PublicKey_URL)
	return &core.OSSCallbackResp{Status:"OK"}, nil
}

func (s *SrvCoreDataStoreServiceServer) QueryFileAccessToken(ctx context.Context, req *core.ReqFileUrl) (*core.ReqFileUrl, error) {
	panic("implement me")
}