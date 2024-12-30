package api

import (
	"github.com/benji-bou/lugh/helper"
	"github.com/labstack/echo/v4/middleware"
)

type Ctrler interface {
	Route() []helper.SrvOption
}

func Listen(ctrl ...Ctrler) error {

	optSrv := make([]helper.SrvOption, 0)
	for _, c := range ctrl {
		optSrv = append(optSrv, c.Route()...)
	}

	return helper.RunServer(
		append([]helper.SrvOption{
			helper.WithMiddleware(middleware.CORS()),
		}, optSrv...)...)

}
