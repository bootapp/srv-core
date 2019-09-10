package server

import (
	"context"
	"encoding/json"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/bootapp/srv-core/proto/core"
	"github.com/bootapp/srv-core/utils"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"strings"
	"time"
)

type SrvCoreSecurityServiceServer struct {
	dalCoreUserClient core.DalUserServiceClient
	dalCoreUserConn *grpc.ClientConn
	aliClient *sdk.Client
}

func NewSecurityServer(dalCoreUserAddr string) *SrvCoreSecurityServiceServer {
	s := &SrvCoreSecurityServiceServer{}
	var err error
	s.aliClient, err = sdk.NewClientWithAccessKey("cn-hangzhou", "", "")
	if err != nil {
		panic(err)
	}
	s.dalCoreUserConn, err = grpc.Dial(dalCoreUserAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	s.dalCoreUserClient = core.NewDalUserServiceClient(s.dalCoreUserConn)

	if err != nil {
		log.Fatalf("phone regex creation error: %v", err)
	}
	return s
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
	if req.Lang != core.SmsType_SMS_LANG_CN {
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
	request.QueryParams["RegionId"] = "cn-hangzhou"
	if !strings.Contains(req.Phone, "-") {
		return nil, status.Error(codes.InvalidArgument, "INVALID_ARG:phone")
	}
	request.QueryParams["PhoneNumbers"] = strings.Split(req.Phone, "-")[1]
	request.QueryParams["SignName"] = "易HS"
	switch req.Type {
	case core.SmsType_SMS_CODE_REGISTER:
		request.QueryParams["TemplateCode"] = "SMS_167875054"
	case core.SmsType_SMS_CODE_LOGIN:
		request.QueryParams["TemplateCode"] = "SMS_167875054"
	case core.SmsType_SMS_CODE_RESET_PASS:
		request.QueryParams["TemplateCode"] = "SMS_168250184"

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