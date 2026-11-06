package mainwindow

import (
	"github.com/lmorg/ttyphoon/utils/swagger"
)

// SwaggerRequest executes an HTTP request for a Swagger/OpenAPI endpoint.
// All request logic lives in utils/swagger; this method is a thin binding.
func (a *WApp) SwaggerRequest(req swagger.RequestT) swagger.ResponseT {
	return swagger.Execute(a.ctx, req)
}
