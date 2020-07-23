package middleware

import (
	"net/http"

	accounts "github.com/owncloud/ocis-accounts/pkg/proto/v0"
	"github.com/owncloud/ocis-pkg/v2/log"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	Logger          log.Logger
	AuthMiddleware  func(http.Handler) http.Handler
	AccountsService accounts.AccountsService
}

// newOptions initializes the available default options.
func newOptions(opts ...Option) Options {
	opt := Options{}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// Logger provides a function to set the logger option.
func Logger(val log.Logger) Option {
	return func(o *Options) {
		o.Logger = val
	}
}

// AuthMiddleware provides a function to set the middleware option.
func AuthMiddleware(val func(http.Handler) http.Handler) Option {
	return func(o *Options) {
		o.AuthMiddleware = val
	}
}

// AccountsService provides an AccountsService client to set the AccountsService option.
func AccountsService(val accounts.AccountsService) Option {
	return func(o *Options) {
		o.AccountsService = val
	}
}
