package server

import (
	"context"
	core "github.com/bootapp/srv-core/proto"
	"github.com/bootapp/srv-core/utils"
	"google.golang.org/grpc"
)

type SrvCoreDataStoreServiceServer struct {
	dalCoreUserClient core.DalUserServiceClient
	dalCoreUserConn *grpc.ClientConn
}

func NewDataStoreServer() *SrvCoreDataStoreServiceServer {
	s := &SrvCoreDataStoreServiceServer{}
	return s
}
func (s *SrvCoreDataStoreServiceServer) QueryUploadToken(ctx context.Context, req *core.UploadTokenReq) (*core.UploadTokenResp, error) {
	var token *core.OSSPolicyToken
	switch req.Type {
	case core.FileDirType_FILE_TYPE_BUSINESS_LICENSE:
		token = utils.GetPolicyToken("business-licenses/", true)
	case core.FileDirType_FILE_TYPE_USER_PROFILE:
		token = utils.GetPolicyToken("profiles/", false)
	case core.FileDirType_FILE_TYPE_INSTITUTE_LOGO:
		token = utils.GetPolicyToken("logos/", false)

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