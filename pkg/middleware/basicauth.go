package middleware

import (
	"context"
	"fmt"
	"net/http"

	accounts "github.com/owncloud/ocis-accounts/pkg/proto/v0"
	"github.com/owncloud/ocis-graph/pkg/service/v0/errorcode"
)

type key int

// CtxUserAccountKey is used to store the record of the authenticated user
const CtxUserAccountKey key = iota

// BasicAuth provides a middleware to check access secured using basic auth
func BasicAuth(opts ...Option) func(http.Handler) http.Handler {
	options := newOptions(opts...)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if username, password, ok := r.BasicAuth(); ok {
				a, err := options.AccountsService.ListAccounts(r.Context(), &accounts.ListAccountsRequest{
					//Query: fmt.Sprintf("username eq '%s'", username),
					// TODO this allows lookung up users when you know the username using basic auth
					// adding the password to the query is an option but sending the sover the wira a la scim seems ugly
					// but to set passwords our accounts need it anyway
					Query: fmt.Sprintf("login eq '%s' and password eq '%s'", username, password),
				})
				if err != nil {
					options.Logger.Info().Err(err).Str("username", username).Msg("Failed to read user")
					w.Header().Add("WWW-Authenticate", "Basic")  // TODO add realm?
					w.Header().Add("WWW-Authenticate", "Bearer") // TODO add realm?
					errorcode.ItemNotFound.Render(w, r, http.StatusUnauthorized)
					return
				}

				ctx := context.WithValue(r.Context(), CtxUserAccountKey, a)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				options.AuthMiddleware(next).ServeHTTP(w, r)
				// TODO add WWW-Authenticate basic auth response header if auth fails
			}
		})
	}
}
