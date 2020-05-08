package middleware

import (
	"context"
	"net/http"

	accounts "github.com/owncloud/ocis-accounts/pkg/proto/v0"
	"github.com/owncloud/ocis-graph/pkg/service/v0/errorcode"
)

type key int

// CtxUserRecordKey is used to store the record of the authenticated user
const CtxUserRecordKey key = iota

// BasicAuth provides a middleware to check access secured using basic auth
func BasicAuth(opts ...Option) func(http.Handler) http.Handler {
	options := newOptions(opts...)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if username, password, ok := r.BasicAuth(); ok {
				record, err := options.AccountsService.Get(r.Context(), &accounts.GetRequest{
					Username: username,
					Password: password,
				})
				if err != nil {
					options.Logger.Info().Err(err).Str("username", username).Msg("Failed to read user")
					w.Header().Add("WWW-Authenticate", "Basic")  // TODO add realm?
					w.Header().Add("WWW-Authenticate", "Bearer") // TODO add realm?
					errorcode.ItemNotFound.Render(w, r, http.StatusUnauthorized)
					return
				}

				ctx := context.WithValue(r.Context(), CtxUserRecordKey, record)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				options.AuthMiddleware(next).ServeHTTP(w, r)
				// TODO add WWW-Authenticate basic auth response header if auth fails
			}
		})
	}
}
