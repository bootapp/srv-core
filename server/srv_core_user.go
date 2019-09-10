package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/bootapp/rest-grpc-oauth2/auth"
	"github.com/bootapp/srv-core/proto/core"
	"github.com/bootapp/srv-core/utils"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"strconv"
	"strings"
)

type SrvCoreUserServiceServer struct {
	dalCoreUserClient core.DalUserServiceClient
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
	s.dalCoreUserClient = core.NewDalUserServiceClient(s.dalCoreUserConn)
	s.auth = auth.GetInstance()
	return s
}

func (s *SrvCoreUserServiceServer) close() {
	err :=s.dalCoreUserConn.Close()
	if err != nil {
		glog.Error(err)
	}
}

func (s *SrvCoreUserServiceServer) Register(ctx context.Context, req *core.RegisterReq) (*core.UserWithOrg, error) {
	glog.Info("registering new user...")
	user := &core.User{}
	switch req.Type {
	case core.RegisterType_REGISTER_TYPE_USERNAME_PASS: // username + password (activated)
		user.Username = req.Key
		user.Password = req.Secret
		user.Status = core.EntityStatus_ENTITY_STATUS_NORMAL
	case core.RegisterType_REGISTER_TYPE_PHONE_PASS: //phone + password (later activation)
		user.Phone = req.Key
		user.Password = req.Secret
		user.Status = core.EntityStatus_ENTITY_STATUS_INACTIVATED
	case core.RegisterType_REGISTER_TYPE_PHONE_CODE: //phone + code (activated)
		err := utils.CheckPhoneCode(core.SmsType_SMS_CODE_REGISTER.String(), req.Key, req.Code)
		if err != nil {
			return nil, err
		}
		user.Phone = req.Key
		user.Password = utils.RandString(10)
		user.Status = core.EntityStatus_ENTITY_STATUS_NORMAL
	case core.RegisterType_REGISTER_TYPE_EMAIL_PASS: //email + pass (later activation)
		user.Email = req.Key
		user.Password = req.Secret
		user.Status = core.EntityStatus_ENTITY_STATUS_INACTIVATED
	case core.RegisterType_REGISTER_TYPE_PHONE_PASS_CODE: //phone + pass + code
		err := utils.CheckPhoneCode(core.SmsType_SMS_CODE_REGISTER.String(), req.Key, req.Code)
		if err != nil {
			return nil, err
		}
		user.Phone = req.Key
		user.Password = req.Secret
		user.Status = core.EntityStatus_ENTITY_STATUS_NORMAL
	}
	_, err := s.dalCoreUserClient.CreateUser(ctx, user)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	return &core.UserWithOrg{}, nil
}
func (s *SrvCoreUserServiceServer) Login(ctx context.Context, req *core.LoginReq) (*core.UserWithOrg, error) {
	glog.Info("user logging in...")
	at, rt, err := s.auth.UserGetAccessToken(req.Type.String(), req.Key, req.Secret, req.Code, req.OrgId)
	if err != nil {
		glog.Error(err)
		return nil, err
	} else if rt == "" || at == "" {
		glog.Error("unexpected error")
		return nil, status.Error(codes.Internal, "INTERNAL:unexpected error when getting access token.")
	} else {
		glog.Info("injecting tokens to cookie...")
		auth.ResponseTokenInjector(ctx, at, rt)
	}
	tokenInfo := strings.Split(at, ".")
	res , err := base64.RawStdEncoding.DecodeString(tokenInfo[1])
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	var personFromJSON interface{}

	decoder := json.NewDecoder(bytes.NewReader(res))
	decoder.UseNumber()
	err = decoder.Decode(&personFromJSON)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	r := personFromJSON.(map[string]interface{})
	userId, err := r["user_id"].(json.Number).Int64()
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	orgIdNum, err := strconv.ParseInt(req.OrgId, 10, 64)
	if err != nil {
		orgIdNum = 0
	}
	qResp, err := s.dalCoreUserClient.ReadUser(ctx, &core.User{Id: userId, OrgId:orgIdNum})
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return qResp, nil
}

func (s *SrvCoreUserServiceServer) ResetPassword(ctx context.Context, req *core.ResetPasswordReq) (*core.Empty, error) {
	updateUserReq := &core.UpdateUserReq{}
	switch req.Type {
	case core.ResetPasswordType_RESET_PASS_TYPE_PHONE_CODE:
		val, err := utils.GetKey(core.SmsType_SMS_CODE_RESET_PASS.String() + req.Key)
		if err != nil {
			return nil, err
		}
		if val != req.Code {
			return nil, status.Error(codes.InvalidArgument, "INVALID_CODE")
		}
		updateUserReq.Type = core.UpdateUserType_UPDATE_USER_TYPE_PHONE
		updateUserReq.User = &core.User{Phone:req.Key, Password:req.Secret}
	}
	_, err := s.dalCoreUserClient.UpdateUser(ctx, updateUserReq)
	if err != nil {
		return nil, err
	}
	return &core.Empty{}, nil
}

func (s *SrvCoreUserServiceServer) Activate(context.Context, *core.Empty) (*core.Empty, error) {
	panic("implement me")
}

func (s *SrvCoreUserServiceServer) UserInfo(context.Context, *core.Empty) (*core.Empty, error) {
	panic("implement me")
}

func (s *SrvCoreUserServiceServer) UpdateUser(context.Context, *core.Empty) (*core.Empty, error) {
	panic("implement me")
}