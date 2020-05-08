package svc

import (
	"net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/go-chi/chi"
	accounts "github.com/owncloud/ocis-accounts/pkg/proto/v0"
	"github.com/owncloud/ocis-graph/pkg/config"
	"github.com/owncloud/ocis-graph/pkg/cs3"
	"github.com/owncloud/ocis-pkg/v2/log"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// Graph defines implements the business logic for Service.
type Graph struct {
	config *config.Config
	mux    *chi.Mux
	logger *log.Logger
	as     accounts.AccountsService
}

// ServeHTTP implements the Service interface.
func (g Graph) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

// GetClient returns a gateway client to talk to reva
func (g Graph) GetClient() (gateway.GatewayAPIClient, error) {
	return cs3.GetGatewayServiceClient(g.config.Reva.Address)
}

type listResponse struct {
	Value interface{} `json:"value,omitempty"`
}

func createUserModelFromRecord(record *accounts.Record) *msgraph.User {
	u := &msgraph.User{
		DirectoryObject: msgraph.DirectoryObject{
			Entity: msgraph.Entity{
				ID: &record.Key,
			},
		},
	}
	if record.Payload != nil && record.Payload.Account != nil && record.Payload.Account.StandardClaims != nil {
		u.DisplayName = &record.Payload.Account.StandardClaims.Name
		u.GivenName = &record.Payload.Account.StandardClaims.GivenName
		u.Mail = &record.Payload.Account.StandardClaims.Email
		u.Surname = &record.Payload.Account.StandardClaims.FamilyName
	}
	return u
}
