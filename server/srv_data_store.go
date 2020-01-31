package server

import (
	"context"
	"github.com/bootapp/rest-grpc-oauth2/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	core "srv-core/proto"
	"srv-core/utils"
)

type SrvCoreDataStoreServiceServer struct {
	dalCoreUserClient core.DalUserServiceClient
	dalCoreUserConn *grpc.ClientConn
	auth *auth.StatelessAuthenticator
}

func NewDataStoreServer() *SrvCoreDataStoreServiceServer {
	s := &SrvCoreDataStoreServiceServer{}
	s.auth = auth.GetInstance()
	return s
}
func (s *SrvCoreDataStoreServiceServer) QueryUploadToken(ctx context.Context, req *core.UploadTokenReq) (*core.UploadTokenResp, error) {
	userId, _ := s.auth.GetAuthInfo(ctx)
	if userId == 0 {
		return nil, status.Error(codes.Unauthenticated, "")
	}
	if req.Type == core.FileDirType_FILE_TYPE_NULL {
		return nil, status.Error(codes.InvalidArgument, "INVALID_ARG:type")
	}
	var token *core.OSSPolicyToken
	switch req.Type {
	case core.FileDirType_FILE_TYPE_BUSINESS_LICENSE:
		token = utils.GetPolicyToken("business-licenses/", true)
	case core.FileDirType_FILE_TYPE_USER_PROFILE:
		token = utils.GetPolicyToken("profiles/", false)
	case core.FileDirType_FILE_TYPE_INSTITUTE_LOGO:
		token = utils.GetPolicyToken("logos/", false)
	case core.FileDirType_FILE_TYPE_TIMETABLE:
		token = utils.GetPolicyToken("timetables/", false)
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