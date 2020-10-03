package routers

import (
	"net/http"
	"z-notes/logging"
)

//LogMiddleware provides verbose logging on all requests
func LogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		TemplateInput := getNewTemplateInput(responseWriter, request)
		logging.WriteLog(logging.LogLevelVerbose, "loggingmiddleware/LogMiddleware", TemplateInput.UserInformation.GetCompositeID(), logging.ResultInfo, []string{request.RequestURI})
		next.ServeHTTP(responseWriter, request) // call ServeHTTP on the original handler
	})
}
