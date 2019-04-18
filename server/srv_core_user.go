package server

import (
	"context"
	"github.com/bootapp/rest-grpc-oauth2/auth"
	"github.com/bootapp/srv-core/proto/clients/dal-core"
	srv "github.com/bootapp/srv-core/proto/server"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
)

type SrvCoreUserServiceServer struct {
	dalCoreUserClient dal_core.DalCoreUserServiceClient
	dalCoreUserConn *grpc.ClientConn
	auth *auth.StatelessAuthenticator
}

func NewSrvCoreUserServiceServer(dalCoreUserAddr string) *SrvCoreUserServiceServer {
	var err error
	s := &SrvCoreUserServiceServer{}
	s.dalCoreUserConn, err = grpc.Dial(dalCoreUserAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	s.dalCoreUserClient = dal_core.NewDalCoreUserServiceClient(s.dalCoreUserConn)
	s.auth = auth.GetInstance()
	return s
}

func (s *SrvCoreUserServiceServer) close() {
	err :=s.dalCoreUserConn.Close()
	if err != nil {
		glog.Error(err)
	}
}

func (s *SrvCoreUserServiceServer) Register(ctx context.Context, req *srv.RegisterReq) (*srv.Resp, error) {
	glog.Info("registering new user...")
	user := &dal_core.UserInfo{}
	switch req.Type {
	case srv.UserServiceType_REGISTER_TYPE_USERNAME_PASS:
		user.Username = req.Key
		user.Password = req.Secret
	}
	resp, err := s.dalCoreUserClient.InvokeNewUser(ctx, user)
	if err != nil {
		glog.Error(err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	switch resp.Status {
	case dal_core.UserServiceType_RESP_SUCCESS:
		return &srv.Resp{}, nil
	case dal_core.UserServiceType_NEW_USER_ERR_DUPLICATE_ENTRY:
		glog.Error("duplicated user")
		return nil, status.Error(codes.InvalidArgument, srv.UserServiceType_REGISTER_ERR_DUPLICATE_ENTRY.String())
	default:
		glog.Error("unexpected error: ", resp.Message)
		return nil, status.Error(codes.Internal, resp.Message)
	}
}
func (s *SrvCoreUserServiceServer) Login(ctx context.Context, req *srv.LoginReq) (resp *srv.LoginResp, err error) {
	glog.Info("user logging in...")
	resp = &srv.LoginResp{}
	at, rt, err := s.auth.UserGetAccessToken(req.Type.String(), req.Key, req.Secret, req.Code, req.OrgId)
	if err != nil {
		switch err.Error() {
		case dal_core.UserServiceType_QUERY_USER_ERR_WRONG_PASS.String():
			err = status.Error(codes.InvalidArgument, srv.UserServiceType_LOGIN_ERR_WRONG_PASS.String())
		default:
			err = status.Error(codes.InvalidArgument, err.Error())
		}
		glog.Error(err)
		return
	} else if rt == "" || at == "" {
		err = status.Error(codes.Internal, srv.UserServiceType_ERR_UNEXPECTED.String())
		glog.Error("unexpected error")
		return
	} else {
		glog.Info("injecting tokens to cookie...")
		auth.ResponseTokenInjector(ctx, at, rt)
		return
	}
}
func (s *SrvCoreUserServiceServer) Activate(context.Context, *srv.Req) (*srv.Resp, error) {
	return nil, nil
}
func (s *SrvCoreUserServiceServer) UserInfo(context.Context, *srv.Req) (*srv.Resp, error) {
	return nil, nil
}
