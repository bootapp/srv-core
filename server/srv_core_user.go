package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/bootapp/rest-grpc-oauth2/auth"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"srv-core/oauth"
	core "srv-core/proto"
	"srv-core/settings"
	"srv-core/utils"
	"strconv"
	"strings"
	"time"
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
// ===================================================== User actions
func (s *SrvCoreUserServiceServer) Register(ctx context.Context, req *core.RegisterReq) (*core.LoginResp, error) {
	ipStr := s.auth.GetRemoteAddr(ctx)
	var ip int32 = 0
	if !strings.Contains(ipStr, "[") && strings.Contains(ipStr, ":"){
		ipStr = strings.Split(ipStr, ":")[0]
		ip = utils.StringIpToInt(ipStr)
	}
	glog.Info("registering new user...")
	if req.Key == "" {
		return nil, status.Error(codes.InvalidArgument, "INVALID_ARG:key")
	}
	user := &core.UserEdit{}
	user.RegIp = &wrappers.Int32Value{Value:ip}
	resp := &core.LoginResp{}

	// ------------ decode base64
	if req.Secret != "" {
		decoded, err := base64.StdEncoding.DecodeString(req.Secret)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "INVALID_ARG:secret")
		}
		req.Secret = string(decoded)
	}

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
						User:&core.UserEdit{
							Email:&wrappers.StringValue{Value:kv[1]},
							Password:&wrappers.StringValue{Value:req.Secret},
							Status:core.EntityStatus_ENTITY_STATUS_NORMAL,
						}})
					if err != nil {
						return nil, err
					}
					userRes, err := s.dalCoreUserClient.ReadUserAuth(ctx, &core.ReadUserReq{Email:kv[1]})
					if err != nil {
						return nil, err
					}
					resp.User = userRes.User
					return resp, nil
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
	resp.User = userWithOrgs.User
	return resp, nil
}
func (s *SrvCoreUserServiceServer) Login(ctx context.Context, req *core.LoginReq) (*core.LoginResp, error) {
	ipStr := s.auth.GetRemoteAddr(ctx)
	var ip int32 = 0
	if !strings.Contains(ipStr, "[") && strings.Contains(ipStr, ":"){
		ipStr = strings.Split(ipStr, ":")[0]
		ip = utils.StringIpToInt(ipStr)
	}
	glog.Info("user logging in...")
	resp := &core.LoginResp{}

	// ------------ decode base64
	if req.Secret != "" {
		decoded, err := base64.StdEncoding.DecodeString(req.Secret)
		if err != nil {
			return nil, err
		}
		req.Secret = string(decoded)
	}

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
		resp.AccessToken = at
		resp.RefreshToken = rt
	}
	// 读取用户信息
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
	userReq := &core.ReadUserReq{Id: userId, OrgId:req.OrgId}
	qResp, err := s.dalCoreUserClient.ReadUserAuth(ctx, userReq)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	if qResp.User.Status == core.EntityStatus_ENTITY_STATUS_FROZEN {
		return nil, status.Error(codes.PermissionDenied, "FROZEN")
	}
	resp.User = qResp.User
	resp.OrgInfo = qResp.OrgInfo
	resp.User.LastLoginIp = ip
	resp.User.LastLoginTime = time.Now().UnixNano()/1e6
	_, _ = s.dalCoreUserClient.UpdateUser(ctx, &core.UpdateUserReq{
		Type:core.UpdateUserType_UPDATE_USER_TYPE_ID,
		User:&core.UserEdit{Id:qResp.User.Id, LastLoginIp:&wrappers.Int32Value{Value:ip},
		LastLoginTime: &wrappers.Int64Value{Value:resp.User.LastLoginTime},
	}})
	return resp, nil
}
func (s *SrvCoreUserServiceServer) Refresh(ctx context.Context, req *core.RefreshReq) (*core.RefreshResp, error) {
	at, rt, err := s.auth.UserRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, err
	}
	if at == "" || rt == "" {
		return nil, status.Error(codes.InvalidArgument, "INVALID_ARG:refresh_token")
	}
	return &core.RefreshResp{AccessToken:at, RefreshToken:rt}, nil
}
func (s *SrvCoreUserServiceServer) ResetPassword(ctx context.Context, req *core.ResetPasswordReq) (*core.Empty, error) {
	// ------------ decode base64
	if req.Secret != "" {
		decoded, err := base64.StdEncoding.DecodeString(req.Secret)
		if err != nil {
			return nil, err
		}
		req.Secret = string(decoded)
	}

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
		updateUserReq.User = &core.UserEdit{Phone:&wrappers.StringValue{Value:req.Key}, Password:&wrappers.StringValue{Value:req.Secret}}
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
// ===================================================== User managements
func (s *SrvCoreUserServiceServer) QueryUsers(ctx context.Context, req *core.QueryUsersReq) (*core.UsersResp, error) {
	userId, orgId, err := s.auth.CheckAuthority(ctx, "P_SYS_USER_R")
	if err != nil {
		return nil, err
	}
	usersReq := &core.ReadUsersReq{UserId:userId, OrgId:orgId,
		Email:req.Email, Username:req.Username, Phone: req.Phone,
		Pagination:req.Pagination}
	return s.dalCoreUserClient.ReadUsers(ctx, usersReq)
}
func (s *SrvCoreUserServiceServer) UpdateUser(ctx context.Context, req *core.UserEdit) (*core.Empty, error) {
	userId, orgId, err := s.auth.CheckAuthority(ctx, "P_SYS_USER_W")
	if err != nil {
		return nil, err
	}
	return s.dalCoreUserClient.UpdateUser(ctx, &core.UpdateUserReq{UserId:userId, OrgId:orgId, User:req})
}
// ===================================================== DictTree
func (s *SrvCoreUserServiceServer) AdminInvokeUpdateDictItem(ctx context.Context, req *core.InvokeUpdateDictReq) (*core.Empty, error) {
	userId, orgId, err := s.auth.CheckAuthority(ctx, "P_SUPER_ADMIN")
	if err != nil {
		return nil, err
	}
	return s.dalCoreUserClient.UpdateDictItems(ctx, &core.DictItemsReq{UserId:userId, OrgId:orgId, Data:[]*core.DictItemEdit{req.Item}})
}
func (s *SrvCoreUserServiceServer) AdminInvokeDeleteDictItem(ctx context.Context, req *core.IdReq) (*core.Empty, error) {
	userId, orgId, err := s.auth.CheckAuthority(ctx, "P_SUPER_ADMIN")
	if err != nil {
		return nil, err
	}
	return s.dalCoreUserClient.DeleteDictItems(ctx, &core.AuthorizedIdsReq{UserId:userId, OrgId:orgId, Ids:[]int64{req.Id}})
}
func (s *SrvCoreUserServiceServer) QueryDictTree(ctx context.Context, req *core.QueryDictTreeReq) (*core.DictItemList, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	return s.dalCoreUserClient.ReadDictItems(ctx, &core.ReadDictItemsReq{UserId:userId, OrgId:orgId,
		Ids:req.Ids, Pid:req.Pid, Status:req.Status, Pagination:req.Pagination})
}
// ===================================================== Feedback
func (s *SrvCoreUserServiceServer) InvokeUpdateFeedback(ctx context.Context, req *core.FeedbackEdit) (*core.Empty, error) {
	userId, orgId, err := s.auth.CheckAuthority(ctx, "P_SUPER_ADMIN")
	if userId == 0 {
		return nil, status.Error(codes.Unauthenticated, "")
	}
	if err != nil {
		if req.Status == core.EntityStatus_ENTITY_STATUS_DONE {
			return nil, status.Error(codes.PermissionDenied, "")
		}
		if req.Reply != nil {
			return nil, status.Error(codes.PermissionDenied, "")
		}
		if req.Id != 0 {
			resp, err := s.dalCoreUserClient.ReadFeedback(ctx, &core.ReadFeedbackReq{UserId:userId, OrgId:orgId,
				Id:req.Id,
				})
			if err != nil {
				return nil, err
			}
			if len(resp.Data) != 1 {
				return nil, status.Error(codes.NotFound, "")
			}
			if resp.Data[0].UserId != userId {
				return nil, status.Error(codes.PermissionDenied, "")
			}
		}
		req.UserId = userId
	}
	return s.dalCoreUserClient.UpdateFeedback(ctx, &core.FeedbackReq{UserId:userId, OrgId:orgId,
		Item:req,
		})
}
func (s *SrvCoreUserServiceServer) QueryFeedback(ctx context.Context, req *core.QueryFeedbackReq) (*core.FeedbackList, error) {
	userId, orgId, err := s.auth.CheckAuthority(ctx, "P_SUPER_ADMIN")
	if err != nil {
		return nil, err
	}
	return s.dalCoreUserClient.ReadFeedback(ctx, &core.ReadFeedbackReq{UserId:userId, OrgId:orgId,
		Status:req.Status,
		})
}
func (s *SrvCoreUserServiceServer) InvokeNewMessage(ctx context.Context, req *core.MessageEdit) (*core.Empty, error) {
	userId, orgId, err := s.auth.CheckAuthority(ctx, "P_SUPER_ADMIN")
	switch req.Type {
	case core.MessageType_MESSAGE_TYPE_ALL: {
		if err != nil {
			return nil, err
		}
		return s.dalCoreUserClient.UpdateMessage(ctx, &core.MessageReq{UserId:userId, OrgId:orgId,
			Msg:req,
		})
	}
	}

	return nil, status.Error(codes.InvalidArgument, "INVALID_ARG:type")
}
func (s *SrvCoreUserServiceServer) QueryMessages(ctx context.Context, req *core.QueryMessagesReq) (*core.MessageList, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 {
		return nil, status.Error(codes.Unauthenticated, "")
	}
	return s.dalCoreUserClient.ReadMessages(ctx, &core.ReadMessagesReq{UserId:userId, OrgId:orgId,
		FromUserId:userId, Pagination:req.Pagination,
		})
}
func (s *SrvCoreUserServiceServer) QueryInbox(ctx context.Context, req *core.PaginationReq) (*core.MessageList, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 {
		return nil, status.Error(codes.Unauthenticated, "")
	}
	return s.dalCoreUserClient.ReadInbox(ctx, &core.ReadInboxReq{UserId:userId, OrgId:orgId,
		QueryUserId:userId, Pagination:req.Pagination,
		})
}
func (s *SrvCoreUserServiceServer) DeleteInbox(ctx context.Context, req *core.StringReq) (*core.Empty, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 {
		return nil, status.Error(codes.Unauthenticated, "")
	}
	idStrs := strings.Split(req.Query, ",")
	ids := make([]int64, len(idStrs))
	var err error
	for i, v := range idStrs {
		ids[i], err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	return s.dalCoreUserClient.DeleteInbox(ctx, &core.UpdateInboxReq{UserId:userId, OrgId:orgId, Ids:ids})
}

func (s *SrvCoreUserServiceServer) InvokeSetReadInbox(ctx context.Context, req *core.IdReq) (*core.Empty, error) {
	userId, orgId := s.auth.GetAuthInfo(ctx)
	if userId == 0 {
		return nil, status.Error(codes.Unauthenticated, "")
	}
	return s.dalCoreUserClient.UpdateInbox(ctx, &core.UpdateInboxReq{UserId:userId, OrgId:orgId,
		Ids:[]int64{req.Id}, Status:core.EntityStatus_ENTITY_STATUS_DONE,
		})
}