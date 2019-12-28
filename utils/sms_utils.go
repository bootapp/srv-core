package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/axgle/mahonia"
	"github.com/bootapp/srv-core/proto"
	"github.com/bootapp/srv-core/settings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)
// =============================== 接口
type SmsUtils interface {
	Init()
	Send(phone, code string, sendType proto.SmsType) error
}

// =============================== 阿里云
type AliSmsUtils struct {
	aliClient *sdk.Client
}
func (s *AliSmsUtils) Init() {
	var err error
	s.aliClient, err = sdk.NewClientWithAccessKey(settings.CredentialAliSMSRegionId, settings.CredentialAliSMSAccessKeyId, settings.CredentialAliSMSAccessSecret)
	if err != nil {
		panic(err)
	}
}
func (s *AliSmsUtils) Send(phone, code string, sendType proto.SmsType) error {
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.Scheme = "https"
	request.Domain = "dysmsapi.aliyuncs.com"
	request.Version = "2017-05-25"
	request.ApiName = "SendSms"
	request.QueryParams["RegionId"] = settings.CredentialAliSMSRegionId
	if !strings.Contains(phone, "-") {
		return status.Error(codes.InvalidArgument, "INVALID_ARG:phone")
	}
	request.QueryParams["PhoneNumbers"] = phone
	request.QueryParams["SignName"] = settings.CredentialAliSMSSignName
	switch sendType {
	case proto.SmsType_SMS_CODE_REGISTER:
		request.QueryParams["TemplateCode"] = settings.CredentialAliSMSRegisterTemplateCode
	case proto.SmsType_SMS_CODE_LOGIN:
		request.QueryParams["TemplateCode"] = settings.CredentialAliSMSLoginTemplateCode
	case proto.SmsType_SMS_CODE_RESET_PASS:
		request.QueryParams["TemplateCode"] = settings.CredentialAliSMSResetPassTemplateCode
	}
	request.QueryParams["TemplateParam"] = "{\"code\":\"" + code + "\"}"
	resp, err := s.aliClient.ProcessCommonRequest(request)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	jsonObj := make(map[string]interface{})
	err = json.Unmarshal(resp.GetHttpContentBytes(), &jsonObj)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if jsonObj["Code"] != "OK" {
		return status.Error(codes.InvalidArgument, jsonObj["Message"].(string))
	}
	return nil
}

// =============================== 梦网
type MonSmsUtils struct {
	enc mahonia.Encoder

}
func (s *MonSmsUtils) Init() {
	s.enc = mahonia.NewEncoder("gbk")
}
type MonSmsResult struct {
	Result int64 `json:"result"`
	MsgId int `json:"msgid"`
	CustId string `json:"custid"`
}
type MonSmsReq struct {
	ApiKey string `json:"apikey"`
	Mobile string `json:"mobile"`
	Content string `json:"content"`
	SrvType string `json:"srvtype"`
}
func (s *MonSmsUtils) Send(phone, code string, sendType proto.SmsType) error {
	gbk := s.enc.ConvertString(fmt.Sprintf("您的验证码是%s，在10分钟内有效。如非本人操作请忽略本短信。", code))
	v := url.Values{}
	v.Set("aa", gbk)
	str := v.Encode()
	arr := strings.Split(str, "=")
	monReq := &MonSmsReq{ApiKey:settings.CredentialMonSMSAPIKey, Mobile:phone, Content:arr[1], SrvType:"本硕岛"}
	jsonBytes, err := json.Marshal(monReq)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	resp, err := http.Post(settings.CredentialMonSMSEndpoint, "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	res := &MonSmsResult{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return err
	}
	if res.Result != 0 {
		return status.Error(codes.InvalidArgument, "SMS_ERROR: " + strconv.FormatInt(res.Result, 10))
	}
	return nil
}