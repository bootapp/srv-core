package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/bootapp/rest-grpc-oauth2/auth"
	"github.com/bootapp/srv-core/oauth"
	"github.com/bootapp/srv-core/proto/core"
	"github.com/bootapp/srv-core/settings"
	"github.com/bootapp/srv-core/utils"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/wrappers"
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
	oauthServer *oauth.UserPassOAuthServer
}

func NewSrvCoreUserServiceServer() *SrvCoreUserServiceServer {
	var err error
	s := &SrvCoreUserServiceServer{}
	s.dalCoreUserConn, err = grpc.Dial(settings.DalCoreUserAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	s.dalCoreUserClient = core.NewDalUserServiceClient(s.dalCoreUserConn)
	s.auth = auth.GetInstance()
	s.oauthServer = oauth.GetOauthServer()
	return s
}

func (s *SrvCoreUserServiceServer) close() {
	err :=s.dalCoreUserConn.Close()
	if err != nil {
		glog.Error(err)
	}
}

func (s *SrvCoreUserServiceServer) QueryUsers(ctx context.Context, req *core.QueryUsersReq) (*core.UsersResp, error) {
	if req.User == nil {
		return nil, status.Error(codes.InvalidArgument, "INVALID_ARG:user")
	}
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if orgId != req.User.OrgId {
		return nil, status.Error(codes.PermissionDenied, "PERMISSION_DENIED")
	}
	usersReq := &core.ReadUsersReq{UserId:userId, OrgId:orgId, User:req.User, Pagination:req.Pagination}
	return s.dalCoreUserClient.ReadUsers(ctx, usersReq)
}

func (s *SrvCoreUserServiceServer) AdminQueryUsers(ctx context.Context, req *core.QueryUsersReq) (*core.UsersResp, error) {
	userId, orgId, err := s.auth.CheckAuthority(ctx, "P_SYS_USER_R")
	if err != nil {
		return nil, err
	}
	usersReq := &core.ReadUsersReq{UserId:userId, OrgId:orgId, User:req.User, Pagination:req.Pagination}
	return s.dalCoreUserClient.ReadUsers(ctx, usersReq)
}

func (s *SrvCoreUserServiceServer) Register(ctx context.Context, req *core.RegisterReq) (*core.UserWithOrgAuth, error) {
	glog.Info("registering new user...")
	user := &core.User{}
	switch req.Type {
	case core.RegisterType_REGISTER_TYPE_USERNAME_PASS: // username + password (activated)
		user.Username = &wrappers.StringValue{Value:req.Key}
		user.Password = &wrappers.StringValue{Value:req.Secret}
		user.Status = core.EntityStatus_ENTITY_STATUS_NORMAL
	case core.RegisterType_REGISTER_TYPE_PHONE_PASS: //phone + password (later activation)
		user.Phone = &wrappers.StringValue{Value:req.Key}
		user.Password = &wrappers.StringValue{Value:req.Secret}
		user.Status = core.EntityStatus_ENTITY_STATUS_INACTIVATED
	case core.RegisterType_REGISTER_TYPE_PHONE_CODE: //phone + code (activated)
		err := utils.CheckPhoneCode(core.SmsType_SMS_CODE_REGISTER.String(), req.Key, req.Code)
		if err != nil {
			return nil, err
		}
		user.Phone = &wrappers.StringValue{Value:req.Key}
		user.Password = &wrappers.StringValue{Value:utils.RandString(10)}
		user.Status = core.EntityStatus_ENTITY_STATUS_NORMAL
	case core.RegisterType_REGISTER_TYPE_EMAIL_PASS: //email + pass (later activation)
		user.Email = &wrappers.StringValue{Value:req.Key}
		user.Password = &wrappers.StringValue{Value:req.Secret}
		user.Status = core.EntityStatus_ENTITY_STATUS_INACTIVATED
	case core.RegisterType_REGISTER_TYPE_PHONE_PASS_CODE: //phone + pass + code
		err := utils.CheckPhoneCode(core.SmsType_SMS_CODE_REGISTER.String(), req.Key, req.Code)
		if err != nil {
			return nil, err
		}
		user.Phone = &wrappers.StringValue{Value:req.Key}
		user.Password = &wrappers.StringValue{Value:req.Secret}
		user.Status = core.EntityStatus_ENTITY_STATUS_NORMAL
	case core.RegisterType_REGISTER_TYPE_CIPHER:
		params := strings.Split(s.oauthServer.AESDecrypt(req.Key), "&")
		for _, d := range params {
			kv := strings.Split(d, "=")
			if len(kv) > 0 {
				switch kv[0] {
				case "email":
					_, err := s.dalCoreUserClient.UpdateUser(ctx, &core.UpdateUserReq{Type:core.UpdateUserType_UPDATE_USER_TYPE_EMAIL,
						User:&core.User{
							Email:&wrappers.StringValue{Value:kv[1]},
							Password:&wrappers.StringValue{Value:req.Secret},
							Status:core.EntityStatus_ENTITY_STATUS_NORMAL,
						}})
					if err != nil {
						return nil, err
					}
					return s.dalCoreUserClient.ReadUserAuth(ctx, &core.ReadUserReq{User:&core.User{Email:&wrappers.StringValue{Value:kv[1]}}})
				}
			}
		}
		return nil, status.Error(codes.InvalidArgument, "INVALID_ARG:cipher")
	default:
		return nil, status.Error(codes.InvalidArgument, "INVALID_ARG:type")
	}
	userReq := &core.CreateUserReq{User:user}
	userWithOrgs, err := s.dalCoreUserClient.CreateUser(ctx, userReq)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	return userWithOrgs, nil
}
func (s *SrvCoreUserServiceServer) Login(ctx context.Context, req *core.LoginReq) (*core.UserWithOrgAuth, error) {
	glog.Info("user logging in...")
	at, rt, err := s.auth.UserGetAccessToken(req.Type.String(), req.Key, req.Secret, req.Code, strconv.FormatInt(req.OrgId, 10))
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
	userReq := &core.ReadUserReq{User: &core.User{Id: userId, OrgId:req.OrgId}}
	qResp, err := s.dalCoreUserClient.ReadUserAuth(ctx, userReq)
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
		updateUserReq.User = &core.User{Phone:&wrappers.StringValue{Value:req.Key}, Password:&wrappers.StringValue{Value:req.Secret}}
	}
	_, err := s.dalCoreUserClient.UpdateUser(ctx, updateUserReq)
	if err != nil {
		return nil, err
	}
	return &core.Empty{}, nil
}

func (s *SrvCoreUserServiceServer) Logout(ctx context.Context, req *core.Empty) (*core.Empty, error) {
	auth.ClearToken(ctx)
	return &core.Empty{}, nil
}

func (s *SrvCoreUserServiceServer) Activate(context.Context, *core.Empty) (*core.Empty, error) {
	panic("implement me")
}

func (s *SrvCoreUserServiceServer) UserInfo(ctx context.Context, req *core.Empty) (*core.Empty, error) {
	err := s.auth.CheckAuthentication(ctx)
	return &core.Empty{}, err
}

func (s *SrvCoreUserServiceServer) UpdateUser(context.Context, *core.Empty) (*core.Empty, error) {
	panic("implement me")
}