
package oauth

import (
	"context"
	"encoding/json"
	"github.com/bootapp/rest-grpc-oauth2/auth"
	"github.com/bootapp/srv-core/proto/core"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var orgNameRegex *regexp.Regexp

func procQueryUserResp(resp *core.UserWithOrgAuth) (userID int64, orgID int64, authorities map[int64][]int64, err error) {
	if orgNameRegex == nil {
		orgNameRegex, err = regexp.Compile("[|:]")
	}
	if err != nil {
		return 0, 0, nil, err
	}
	if resp.User == nil || len(resp.OrgInfo) == 0 {
		return 0, 0, nil, status.Error(codes.Internal, "INTERNAL:error implementation of queryUser")
	}
	if len(resp.OrgInfo) > 1 {
		result := ""
		for idx, orgInfo := range resp.OrgInfo {
			if idx == 0 {
				result += strconv.FormatInt(orgInfo.Id, 10) + ":"+ orgNameRegex.ReplaceAllString(orgInfo.Name, "")
			} else {
				result += "|" + strconv.FormatInt(orgInfo.Id, 10) + ":"+ orgNameRegex.ReplaceAllString(orgInfo.Name, "")
			}
		}
		return 0, 0, nil, status.Error(codes.FailedPrecondition, result)
	} else {
		authorities, err = auth.AuthorityEncode(strings.Split(resp.OrgInfo[0].AuthorityGroups, ";"), strings.Split(resp.OrgInfo[0].Authorities, ";"))
		return resp.User.Id, resp.OrgInfo[0].Id, authorities, err
	}

}
func loginHandler(username, password, code, orgId, authType string) (userID int64, orgID int64, authorities map[int64][]int64, err error) {
	glog.Info("authenticating user...")
	orgIdNum, err := strconv.ParseInt(orgId, 10, 64)
	if err != nil {
		orgIdNum = 0
	}
	switch authType {
	case "LOGIN_TYPE_USERNAME_PASS":
		resp, err := dalCoreUserClient.ReadUserAuth(context.Background(), &core.ReadUserReq{User:&core.User{Username: &wrappers.StringValue{Value:username},
			Password: &wrappers.StringValue{Value:password}, OrgId:orgIdNum}})
		if err != nil {
			return 0, 0, nil, err
		}
		return procQueryUserResp(resp)
	case "LOGIN_TYPE_EMAIL_PASS":
		resp, err := dalCoreUserClient.ReadUserAuth(context.Background(), &core.ReadUserReq{User:&core.User{Email: &wrappers.StringValue{Value:username},
			Password: &wrappers.StringValue{Value:password}, OrgId:orgIdNum}})
		if err != nil {
			return 0, 0, nil, err
		}
		return procQueryUserResp(resp)
	case "LOGIN_TYPE_PHONE_PASS":
		resp, err := dalCoreUserClient.ReadUserAuth(context.Background(), &core.ReadUserReq{User:&core.User{Phone: &wrappers.StringValue{Value:username},
			Password: &wrappers.StringValue{Value:password}, OrgId:orgIdNum}})
		if err != nil {
			return 0, 0, nil, err
		}
		return procQueryUserResp(resp)
	case "LOGIN_TYPE_ANY_PASS":
		resp, err := dalCoreUserClient.ReadUserAuth(context.Background(),
			&core.ReadUserReq{User:&core.User{Phone: &wrappers.StringValue{Value:username}, Email: &wrappers.StringValue{Value:username},
				Username:&wrappers.StringValue{Value:username}, Password: &wrappers.StringValue{Value:password}, OrgId:orgIdNum}})
		if err != nil {
			return 0, 0, nil, err
		}
		return procQueryUserResp(resp)
	case "LOGIN_TYPE_PHONE_CODE":
		return 0, 0, nil, status.Error(codes.Unimplemented, "INTERNAL:not implemented yet")
	case "LOGIN_TYPE_PHONE_LOGIN_OR_REG":
		return 0, 0, nil, status.Error(codes.Unimplemented, "INTERNAL:not implemented yet")
	}

	return 0, 0, nil, status.Error(codes.Unimplemented, "INTERNAL:not implemented yet")
}
func ServeOAuthHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/oauth/authorize":
		err := oauthServer.Srv.HandleAuthorizeRequest(w, r)
		if err != nil {
			stat := status.Convert(err)
			http.Error(w, stat.Message(), http.StatusBadRequest)
		}
	case "/api/oauth/token":
		err := oauthServer.Srv.HandleTokenRequest(w, r)
		if err != nil {
			stat := status.Convert(err)
			if stat.Code() == codes.FailedPrecondition {
				http.Error(w, stat.Message(), http.StatusMultipleChoices)
			} else {
				http.Error(w, stat.Message(), http.StatusBadRequest)
			}
		}
	case "/api/oauth/token_key":
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")

		w.WriteHeader(http.StatusOK)
		resp := make(map[string] string)
		resp["alg"] = jwt.SigningMethodRS256.Name
		resp["value"] = string(oauthServer.GetPublicKey())
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			stat := status.Convert(err)
			http.Error(w, stat.Message(), http.StatusBadRequest)
		}
	}

}