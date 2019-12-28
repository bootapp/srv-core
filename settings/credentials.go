package settings

import "time"

var (
	CredentialEmailServerHost = ""
	CredentialEmailServerPort = 465
	CredentialEmailServerMail = ""
	CredentialEmailServerPassword = ""
	CredentialEmailFromEmail = ""
	CredentialEmailFromName = ""
)

var (
	SmsRedisExireTime = 10 * time.Minute

	SmsServiceType = "MONYUN" // "ALIYUN
	CredentialAliSMSRegionId = ""
	CredentialAliSMSAccessKeyId = ""
	CredentialAliSMSAccessSecret = ""
	CredentialAliSMSSignName = ""
	CredentialAliSMSLoginTemplateCode = ""
	CredentialAliSMSRegisterTemplateCode = ""
	CredentialAliSMSResetPassTemplateCode = ""

	CredentialMonSMSEndpoint = ""
	CredentialMonSMSAPIKey = ""
)

var (
	CredentialAliOSSKey = ""
	CredentialAliOSSSecret = ""
	CredentialAliOSSHostPub = ""   //公共读，私有写
	CredentialAliOSSHostSecret = "" //私有读，私有写

	CredentialAliOSSCallbackHost = ""
	CredentialAliOSSExpireTime int64 = 30
)