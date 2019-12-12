package utils

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/bootapp/srv-core/proto/core"
	"github.com/bootapp/srv-core/settings"
	"hash"
	"io"
	"time"
)

func getGmtIso8601(expireEnd int64) string {
	var tokenExpire = time.Unix(expireEnd, 0).Format("2006-01-02T15:04:05Z")
	return tokenExpire
}

type ConfigStruct struct{
	Expiration string `json:"expiration"`
	Conditions [][]string `json:"conditions"`
}

type CallbackParam struct{
	CallbackUrl string `json:"callbackUrl"`
	CallbackBody string `json:"callbackBody"`
	CallbackBodyType string `json:"callbackBodyType"`
}

func GetPolicyToken(uploadDir string) *core.OSSPolicyToken {
	now := time.Now().Unix()
	expire_end := now + settings.CredentialAliOSSExpireTime
	var tokenExpire = getGmtIso8601(expire_end)

	//create post policy json
	var config ConfigStruct
	config.Expiration = tokenExpire
	var condition []string
	condition = append(condition, "starts-with")
	condition = append(condition, "$key")
	condition = append(condition, uploadDir)
	config.Conditions = append(config.Conditions, condition)

	//calucate signature
	result,err:=json.Marshal(config)
	debyte := base64.StdEncoding.EncodeToString(result)
	h := hmac.New(func() hash.Hash { return sha1.New() }, []byte(settings.CredentialAliOSSSecret))
	_, err = io.WriteString(h, debyte)
	signedStr := base64.StdEncoding.EncodeToString(h.Sum(nil))

	var callbackParam CallbackParam
	callbackParam.CallbackUrl = settings.CredentialAliOSSCallbackHost
	callbackParam.CallbackBody = "filename=${object}&size=${size}&mimeType=${mimeType}&height=${imageInfo.height}&width=${imageInfo.width}"
	callbackParam.CallbackBodyType = "application/x-www-form-urlencoded"
	callback_str,err:=json.Marshal(callbackParam)
	if err != nil {
		fmt.Println("callback json err:", err)
	}
	callbackBase64 := base64.StdEncoding.EncodeToString(callback_str)

	policyToken := &core.OSSPolicyToken{}
	policyToken.Key = settings.CredentialAliOSSKey
	policyToken.Host = settings.CredentialAliOSSHost
	policyToken.Expire = expire_end
	policyToken.Signature = string(signedStr)
	policyToken.Dir = uploadDir
	policyToken.Policy = string(debyte)
	policyToken.Callback = string(callbackBase64)
	return policyToken
}