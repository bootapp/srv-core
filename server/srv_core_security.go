package server

import (
	"context"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/gomail.v2"
	"log"
	"srv-core/oauth"
	core "srv-core/proto"
	"srv-core/settings"
	"srv-core/utils"
	"strings"
)

type SrvCoreSecurityServiceServer struct {
	dalCoreUserClient core.DalUserServiceClient
	dalCoreUserConn *grpc.ClientConn
	oauthServer *oauth.UserPassOAuthServer
	smsUtils utils.SmsUtils
}

func NewSecurityServer() *SrvCoreSecurityServiceServer {
	s := &SrvCoreSecurityServiceServer{}
	var err error
	s.dalCoreUserConn, err = grpc.Dial(settings.DalCoreUserAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	s.dalCoreUserClient = core.NewDalUserServiceClient(s.dalCoreUserConn)

	if err != nil {
		log.Fatalf("phone regex creation error: %v", err)
	}
	s.oauthServer = oauth.GetOauthServer()
	switch settings.SmsServiceType {
	case "ALIYUN":
		s.smsUtils = &utils.AliSmsUtils{}
	case "MONYUN":
		s.smsUtils = &utils.MonSmsUtils{}
	}
	s.smsUtils.Init()
	return s
}
func (s *SrvCoreSecurityServiceServer) Cipher(ctx context.Context, req *core.CipherReq) (*core.CipherResp, error) {
	if s.oauthServer == nil {
		s.oauthServer = oauth.GetOauthServer()
	}
	resp := &core.CipherResp{}
	var err error
	switch req.Type {
	case core.CipherType_CIPHER_TYPE_AES_ENCRYPT:
		resp.Data = s.oauthServer.AESEncrypt(req.Data)
	case core.CipherType_CIPHER_TYPE_AES_DECRYPT:
		resp.Data = s.oauthServer.AESDecrypt(req.Data)
	case core.CipherType_CIPHER_TYPE_RS256_SIGN:
		resp.Sig, err = s.oauthServer.RS256Sign(req.Data)
		if err != nil {
			return nil, err
		}
	case core.CipherType_CIPHER_TYPE_RS256_VERIFY:
		resp.Valid = s.oauthServer.RS256Verify(req.Data, req.Sig)
	}
	return resp, nil
}
func (s *SrvCoreSecurityServiceServer) SendPhoneCode(ctx context.Context, req *core.SmsReq) (*core.Empty, error) {
	userReq := &core.User{}
	userReq.Phone = req.Phone
	phoneExists := false
	_, err := s.dalCoreUserClient.VerifyUniqueUser(ctx, userReq)
	if err != nil {
		if status.Convert(err).Code() == codes.AlreadyExists {
			phoneExists = true
		} else {
			glog.Error(err)
			return nil, err
		}
	}
	if req.Lang != core.Language_LANG_ZH_CN {
		return nil, status.Error(codes.InvalidArgument, "INVALID_ARG:lang")
	}
	if req.Type != core.SmsType_SMS_CODE_LOGIN && req.Type != core.SmsType_SMS_CODE_RESET_PASS && req.Type != core.SmsType_SMS_CODE_REGISTER {
		return nil, status.Error(codes.InvalidArgument, "INVALID_ARG:type")
	}
	if (req.Type == core.SmsType_SMS_CODE_LOGIN || req.Type == core.SmsType_SMS_CODE_RESET_PASS) && !phoneExists {
		return nil, status.Error(codes.NotFound, "NON_EXISTS")
	} else if req.Type == core.SmsType_SMS_CODE_REGISTER && phoneExists {
		return nil, status.Error(codes.AlreadyExists, "ALREADY_EXISTS")
	}

	text := utils.GenCode(6)
	err = utils.SetKey(req.Type.String() + req.Phone, text, settings.SmsRedisExireTime)
	if err != nil {
		return nil, err
	}
	// todo: response to different country codes
	phone := strings.Split(req.Phone, "-")[1]
	err = s.smsUtils.Send(phone, text, req.Type)
	if err != nil {
		return nil, err
	}
	return &core.Empty{}, nil
}

func (s *SrvCoreSecurityServiceServer) VerifyPhoneCode(ctx context.Context, req *core.SmsVerifyReq) (*core.Empty, error) {
	if req.Type != core.SmsType_SMS_CODE_LOGIN && req.Type != core.SmsType_SMS_CODE_RESET_PASS && req.Type != core.SmsType_SMS_CODE_REGISTER {
		return nil, status.Error(codes.InvalidArgument, "INVALID_ARG:type")
	}
	val, err := utils.GetKey(req.Type.String() + req.Phone)
	if err != nil {
		return nil, err
	}
	if val != req.Code {
		return nil, status.Error(codes.InvalidArgument, "INVALID_CODE")
	}
	return &core.Empty{}, nil
}

func (s *SrvCoreSecurityServiceServer) SendEmail(ctx context.Context, req *core.SendEmailReq) (*core.Empty, error) {
	m := gomail.NewMessage()
	m.SetHeader("To", req.Email)
	m.SetAddressHeader("From", settings.CredentialEmailFromEmail, settings.CredentialEmailFromName)
	d:= gomail.NewDialer(settings.CredentialEmailServerHost, settings.CredentialEmailServerPort,
		settings.CredentialEmailServerMail, settings.CredentialEmailServerPassword)

	switch req.Type {
	default: // plain
		m.SetHeader("Subject", req.Subject)
		m.SetBody("text/html", req.Content)
	}

	err := d.DialAndSend(m)
	if err != nil {
		return nil, err
	}
	return &core.Empty{}, nil
}

func (s *SrvCoreSecurityServiceServer) VerifyEmailCode(context.Context, *core.VerifyEmailCodeReq) (*core.Empty, error) {
	panic("implement me")
}
