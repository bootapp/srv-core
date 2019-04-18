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
	user := &dal_core.UserInfo{}
	switch req.Type {
	case srv.UserServiceType_REGISTER_REQ_TYPE_USERNAME_PASS:
		user.Username = req.Key
		user.Password = req.Secret
	}
	resp, err := s.dalCoreUserClient.InvokeNewUser(ctx, user)
	if err != nil {
		glog.Error(err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	if resp.Status != dal_core.UserServiceType_RESP_SUCCESS {
		return nil, status.Error(codes.InvalidArgument, resp.Message)
	}
	return &srv.Resp{}, nil
}
func (s *SrvCoreUserServiceServer) Login(context.Context, *srv.LoginReq) (*srv.Resp, error) {
	return nil, nil
}
func (s *SrvCoreUserServiceServer) Activate(context.Context, *srv.Req) (*srv.Resp, error) {
	return nil, nil
}
func (s *SrvCoreUserServiceServer) UserInfo(context.Context, *srv.Req) (*srv.Resp, error) {
	return nil, nil
}
