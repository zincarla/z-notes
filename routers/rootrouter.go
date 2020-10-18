package routers

import (
	"html/template"
	"net/http"
	"path"
	"path/filepath"
	"z-notes/config"
	"z-notes/database"
	"z-notes/logging"
)

//RootRouter serves requests to the root (/)
func RootRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	if TemplateInput.IsLoggedOn() {
		//Get menu root for logged on user
		//Grab root pages so that the menu may be constructed in template
		roots, err := database.DBInterface.GetRootPages(TemplateInput.UserInformation.DBID)
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get root pages", err.Error()})
			TemplateInput.HTMLMessage = template.HTML("Failed to get root pages, internal error occured.")
		} else {
			TemplateInput.PageData.Children = roots
		}
		TemplateInput.BreadCrumbRoot = TemplateInput.PageData
	}
	replyWithTemplate("index.html", TemplateInput, responseWriter, request)
}

//BadConfigRouter is served when the config failed to load
func BadConfigRouter(responseWriter http.ResponseWriter, request *http.Request) {
	logging.WriteLog(logging.LogLevelVerbose, "ContentRouter/BadConfigRouter", "*", logging.ResultSuccess, []string{path.Join(config.Configuration.HTTPRoot, "resources"+string(filepath.Separator)+"updateconfig.html")})
	//Do not cache this file
	//Otherwise can cause headaches once issue is fixed and server is rebooted as client will just reshow config instead of working service
	responseWriter.Header().Add("Cache-Control", "no-cache, private, max-age=0")
	responseWriter.Header().Add("Pragma", "no-cache")
	responseWriter.Header().Add("X-Accel-Expires", "0")
	http.ServeFile(responseWriter, request, path.Join(config.Configuration.HTTPRoot, "resources"+string(filepath.Separator)+"updateconfig.html"))
}
