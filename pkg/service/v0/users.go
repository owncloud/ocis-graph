package svc

import (
	"context"
	"net/http"

	accounts "github.com/owncloud/ocis-accounts/pkg/proto/v0"
	authmw "github.com/owncloud/ocis-graph/pkg/middleware"
	"github.com/owncloud/ocis-graph/pkg/service/v0/errorcode"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/owncloud/ocis-pkg/v2/oidc"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// UserCtx middleware is used to load an User object from
// the URL parameters passed through as the request. In case
// the User could not be found, we stop here and return a 404.
func (g Graph) UserCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		userID := chi.URLParam(r, "userID")
		if userID == "" {
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest)
			return
		}

		record, err := g.as.Get(r.Context(), &accounts.GetRequest{
			Uuid: userID,
		})
		if err != nil {
			g.logger.Info().Err(err).Str("uuid", userID).Msg("Failed to read user")
			errorcode.ItemNotFound.Render(w, r, http.StatusNotFound)
			return
		}

		// store record in context so handlers can access it
		ctx := context.WithValue(r.Context(), authmw.CtxUserRecordKey, record)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (g Graph) getUserRecord(ctx context.Context) (record *accounts.Record, err error) {
	// check if record is already available
	record = ctx.Value(authmw.CtxUserRecordKey).(*accounts.Record)
	if record != nil {
		return
	}

	// check oidc claims
	claims := oidc.FromContext(ctx)
	g.logger.Info().Interface("Claims", claims).Msg("Claims in /me")

	// lookup using sub&iss
	record, err = g.as.Get(ctx, &accounts.GetRequest{
		Identity: &accounts.IdHistory{
			Iss: claims.Iss,
			Sub: claims.Sub,
		},
	})
	if err != nil {
		g.logger.Info().Err(err).Str("iss", claims.Iss).Str("sub", claims.Sub).Msg("Failed to read user")
	}

	// TODO fallback to lookup using email or username
	return
}

// GetMe implements the Service interface.
func (g Graph) GetMe(w http.ResponseWriter, r *http.Request) {

	record, err := g.getUserRecord(r.Context())
	if err != nil {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound)
		return
	}

	me := createUserModelFromRecord(record)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, me)
}

// GetUsers implements the Service interface.
func (g Graph) GetUsers(w http.ResponseWriter, r *http.Request) {

	records, err := g.as.Search(r.Context(), &accounts.Query{})
	if err != nil {
		g.logger.Info().Err(err).Msg("Failed to list users")
		// TODO only return not found if query had a filter?
		// TODO translate errors
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound)
		return
	}

	var users []*msgraph.User

	for _, record := range records.Records {
		users = append(users, createUserModelFromRecord(record))
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &listResponse{Value: users})
}

// GetUser implements the Service interface.
func (g Graph) GetUser(w http.ResponseWriter, r *http.Request) {
	record := r.Context().Value(authmw.CtxUserRecordKey).(*accounts.Record)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, createUserModelFromRecord(record))
}
