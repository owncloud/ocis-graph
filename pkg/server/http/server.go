package http

import (
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/client"
	accounts "github.com/owncloud/ocis-accounts/pkg/proto/v0"
	authmw "github.com/owncloud/ocis-graph/pkg/middleware"
	svc "github.com/owncloud/ocis-graph/pkg/service/v0"
	"github.com/owncloud/ocis-graph/pkg/version"
	"github.com/owncloud/ocis-pkg/v2/middleware"
	"github.com/owncloud/ocis-pkg/v2/oidc"
	"github.com/owncloud/ocis-pkg/v2/service/http"
)

// Server initializes the http service and server.
func Server(opts ...Option) (http.Service, error) {
	options := newOptions(opts...)

	service := http.NewService(
		http.Logger(options.Logger),
		http.Namespace(options.Config.HTTP.Namespace),
		http.Name("graph"),
		http.Version(version.String),
		http.Address(options.Config.HTTP.Addr),
		http.Context(options.Context),
		http.Flags(options.Flags...),
	)

	as, err := getAccountsService()
	if err != nil {
		return service, err
	}

	handle := svc.NewService(
		svc.AccountsService(as),
		svc.Logger(options.Logger),
		svc.Config(options.Config),
		svc.Middleware(
			middleware.RealIP,
			middleware.RequestID,
			middleware.Cache,
			middleware.Cors,
			middleware.Secure,
			middleware.Version(
				"graph",
				version.String,
			),
			middleware.Logger(
				options.Logger,
			),
			authmw.BasicAuth(
				authmw.AuthMiddleware( // wrap with basic auth middleware for ldap bind like requests
					middleware.OpenIDConnect(
						oidc.Endpoint(options.Config.OpenIDConnect.Endpoint),
						oidc.Realm(options.Config.OpenIDConnect.Realm),
						oidc.Insecure(options.Config.OpenIDConnect.Insecure),
						oidc.Logger(options.Logger),
					),
				),
				authmw.AccountsService(as),
				authmw.Logger(
					options.Logger,
				),
			),
		),
	)

	{
		handle = svc.NewInstrument(handle, options.Metrics)
		handle = svc.NewLogging(handle, options.Logger)
		handle = svc.NewTracing(handle)
	}

	service.Handle(
		"/",
		handle,
	)

	service.Init()
	return service, nil
}

// getAccountsService returns an ocis-accounts service
func getAccountsService() (accounts.AccountsService, error) {
	service := micro.NewService()

	// parse command line flags
	service.Init()

	err := service.Client().Init(
		client.ContentType("application/json"),
	)
	if err != nil {
		return nil, err
	}
	return accounts.NewAccountsService("com.owncloud.api.accounts", service.Client()), nil
}
