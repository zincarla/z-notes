package routers

import (
	"html"
	"net/http"
	"path"
	"path/filepath"
	"z-notes/config"
	"z-notes/logging"

	"github.com/gorilla/mux"
)

//ResourceRouter handles requests to /resources
func ResourceRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	logging.WriteLog(logging.LogLevelVerbose, "ContentRouter/ResourceRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultSuccess, []string{"resources" + string(filepath.Separator) + urlVariables["file"]})
	http.ServeFile(responseWriter, request, path.Join(config.Configuration.HTTPRoot, "resources"+string(filepath.Separator)+urlVariables["file"]))
}

//RedirectRouter handles requests to /redirect
func RedirectRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	TemplateInput.RedirectLink = request.FormValue("RedirectLink")
	logging.WriteLog(logging.LogLevelVerbose, "ContentRouter/RedirectRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultInfo, []string{html.EscapeString(request.FormValue("RedirectLink"))})
	replyWithTemplate("redirect.html", TemplateInput, responseWriter, request)
}
