package server

import (
	"context"
	"encoding/json"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/bootapp/srv-core/oauth"
	core "github.com/bootapp/srv-core/proto"
	"github.com/bootapp/srv-core/settings"
	"github.com/bootapp/srv-core/utils"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/gomail.v2"
	"log"
	"strings"
	"time"
)

type SrvCoreSecurityServiceServer struct {
	dalCoreUserClient core.DalUserServiceClient
	dalCoreUserConn *grpc.ClientConn
	aliClient *sdk.Client
	oauthServer *oauth.UserPassOAuthServer
}

func NewSecurityServer() *SrvCoreSecurityServiceServer {
	s := &SrvCoreSecurityServiceServer{}
	var err error
	s.aliClient, err = sdk.NewClientWithAccessKey(settings.CredentialSMSRegionId, settings.CredentialSMSAccessKeyId, settings.CredentialSMSAccessSecret)
	if err != nil {
		panic(err)
	}
	s.dalCoreUserConn, err = grpc.Dial(settings.DalCoreUserAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	s.dalCoreUserClient = core.NewDalUserServiceClient(s.dalCoreUserConn)

	if err != nil {
		log.Fatalf("phone regex creation error: %v", err)
	}
	s.oauthServer = oauth.GetOauthServer()
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
	userReq.Phone = &wrappers.StringValue{Value:req.Phone}
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

	err = utils.SetKey(req.Type.String() + req.Phone, text, 5 * time.Minute)
	if err != nil {
		return nil, err
	}

	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.Scheme = "https"
	request.Domain = "dysmsapi.aliyuncs.com"
	request.Version = "2017-05-25"
	request.ApiName = "SendSms"
	request.QueryParams["RegionId"] = settings.CredentialSMSRegionId
	if !strings.Contains(req.Phone, "-") {
		return nil, status.Error(codes.InvalidArgument, "INVALID_ARG:phone")
	}
	request.QueryParams["PhoneNumbers"] = strings.Split(req.Phone, "-")[1]
	request.QueryParams["SignName"] = settings.CredentialSMSSignName
	switch req.Type {
	case core.SmsType_SMS_CODE_REGISTER:
		request.QueryParams["TemplateCode"] = settings.CredentialSMSRegisterTemplateCode
	case core.SmsType_SMS_CODE_LOGIN:
		request.QueryParams["TemplateCode"] = settings.CredentialSMSLoginTemplateCode
	case core.SmsType_SMS_CODE_RESET_PASS:
		request.QueryParams["TemplateCode"] = settings.CredentialSMSResetPassTemplateCode

	}
	request.QueryParams["TemplateParam"] = "{\"code\":\"" + text + "\"}"
	resp, err := s.aliClient.ProcessCommonRequest(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	jsonObj := make(map[string]interface{})
	err = json.Unmarshal(resp.GetHttpContentBytes(), &jsonObj)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if jsonObj["Code"] != "OK" {
		return nil, status.Error(codes.InvalidArgument, jsonObj["Message"].(string))
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
