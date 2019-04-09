
package oauth

import (
"encoding/json"
"github.com/bootapp/rest-grpc-oauth2/auth"
"github.com/dgrijalva/jwt-go"
"github.com/golang/glog"
"net/http"
)
func loginHandler(username, password string, code string, orgId string, authType string) (userID int64, orgID int64, authorities map[int64]int64, err error) {
	switch authType {
	case "pass":
	case "phone":
	case "email":
		//	return 123,123, auth.AuthorityEncode([]string{"ORG_USER","ORG_DEBIT"}, []string{"AUTH_USER","AUTH_DEBIT_MANAGE"}),nil
	}
	glog.Info("authenticating user...")
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
