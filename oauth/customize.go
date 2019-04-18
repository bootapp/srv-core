
package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/bootapp/rest-grpc-oauth2/auth"
	"github.com/bootapp/srv-core/proto/clients/dal-core"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"net/http"
)

func loginHandler(username, password, code, orgId, authType string) (userID, orgID int64, authorities map[int64]int64, err error) {
	glog.Info("authenticating user...")
	switch authType {
	case "LOGIN_TYPE_USERNAME_PASS":
		resp, err := dalCoreUserClient.QueryUser(context.Background(), &dal_core.UserInfo{Username: username, Password: password})
		if err != nil {
			return 0, 0, nil, err
		} else if resp.Status != dal_core.UserServiceType_RESP_SUCCESS {
			return 0, 0, nil, errors.New(resp.Message)
		}
		return resp.User.Id, resp.User.OrgId, auth.AuthorityEncode([]string{"ORG_AUTH_USER","ORG_AUTH_DEBIT"}, []string{"AUTH_DEBIT_TEST_R","AUTH_USER"}), nil

	case "LOGIN_TYPE_EMAIL_PASS":
	case "LOGIN_TYPE_PHONE_PASS":
	}
	return 1234, 1234, auth.AuthorityEncode([]string{"ORG_AUTH_USER","ORG_AUTH_DEBIT"}, []string{"AUTH_DEBIT_TEST_R","AUTH_USER"}),nil
}

func httpHandlers(server *UserPassOAuthServer) {
	http.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		err := server.Srv.HandleAuthorizeRequest(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		err := server.Srv.HandleTokenRequest(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	http.HandleFunc("/token_key", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")

		status := http.StatusOK
		w.WriteHeader(status)
		resp := make(map[string] string)
		resp["alg"] = jwt.SigningMethodRS256.Name
		resp["value"] = string(server.GetPublicKey())
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})
}
