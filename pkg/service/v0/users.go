package svc

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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

		a, err := g.as.GetAccount(r.Context(), &accounts.GetAccountRequest{
			Id: userID,
		})
		if err != nil {
			// TODO differentiate betwen not found and too many users error
			g.logger.Info().Err(err).Str("userID", userID).Msg("Failed to read user")
			errorcode.ItemNotFound.Render(w, r, http.StatusNotFound)
			return
		}

		// store account in context so handlers can access it
		ctx := context.WithValue(r.Context(), authmw.CtxUserAccountKey, a)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (g Graph) getUserAccount(ctx context.Context) (a *accounts.Account, err error) {
	// check if account is already available
	a = ctx.Value(authmw.CtxUserAccountKey).(*accounts.Account)
	if a != nil {
		return
	}

	// get oidc claims
	claims := oidc.FromContext(ctx)
	g.logger.Debug().Interface("Claims", claims).Msg("Claims in /me")

	// TODO read query from config. use string replace for {mail}, {iss}, {sub} or whatever claims
	query := "mail eq '{mail}'"
	query = strings.ReplaceAll(query, "{mail}", strings.ReplaceAll(claims.Email, "'", "''"))

	// lookup using claims
	var lar *accounts.ListAccountsResponse
	lar, err = g.as.ListAccounts(ctx, &accounts.ListAccountsRequest{Query: query, PageSize: 2})
	if err != nil {
		g.logger.Error().Err(err).Str("iss", claims.Iss).Str("sub", claims.Sub).Msg("Failed to read user")
		return
	}

	switch len(lar.Accounts) {
	case 0:
		err = fmt.Errorf("account not found for %s", query)
		g.logger.Error().Err(err).Msg("Failed to read account")
	case 1:
		a = lar.Accounts[0]
	default:
		err = fmt.Errorf("more than one acount accounts for %s: %+v, %+v", query, lar.Accounts[0], lar.Accounts[1])
		g.logger.Error().Err(err).Msg("Failed to read account")
	}

	return
}

// GetMe implements the Service interface.
func (g Graph) GetMe(w http.ResponseWriter, r *http.Request) {

	a, err := g.getUserAccount(r.Context())
	if err != nil {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound)
		return
	}

	me := createUserModelFromAccount(a)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, me)
}

// GetUsers implements the Service interface.
func (g Graph) GetUsers(w http.ResponseWriter, r *http.Request) {
	var ps int64
	top := r.URL.Query().Get("top")
	if top == "" {
		top = r.URL.Query().Get("$top")
	}
	if top != "" {
		var err error
		ps, err = strconv.ParseInt(top, 10, 32)
		if err != nil {
			g.logger.Info().Err(err).Msg("Failed to parse top")
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest)
			return
		}
	}

	filter := r.URL.Query().Get("filter")
	// fallback to $filter
	if filter == "" {
		filter = r.URL.Query().Get("$filter")
	}
	// TODO parse filter and translate names
	la := &accounts.ListAccountsRequest{
		Query:     filter,
		PageSize:  int32(ps),
		PageToken: r.URL.Query().Get("page_token"),
	}

	lar, err := g.as.ListAccounts(r.Context(), la)
	if err != nil {
		g.logger.Info().Err(err).Msg("Failed to list accounts")
		// TODO only return not found if query had a filter?
		// TODO translate errors
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound)
		return
	}

	var users []*msgraph.User

	for _, a := range lar.Accounts {
		users = append(users, createUserModelFromAccount(a))
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &listResponse{Value: users})
}

// GetUser implements the Service interface.
func (g Graph) GetUser(w http.ResponseWriter, r *http.Request) {
	a := r.Context().Value(authmw.CtxUserAccountKey).(*accounts.Account)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, createUserModelFromAccount(a))
}
