package routers

import (
	"html/template"
	"net/http"
	"os"
	"path"
	"strconv"
	"z-notes/config"
	"z-notes/database"
	"z-notes/interfaces"
	"z-notes/logging"

	"github.com/gorilla/mux"
)

//PageRouter serves requests to /page/{pageID}/view
func PageRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]

	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with 404
		logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "The page requested could not be found", "pageError")
		return
	}

	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	if TemplateInput.IsLoggedOn() {
		//Check user permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with 404
			logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "authError")
			return
		}
	} else {
		access.User.DBID = interfaces.AnonymousUserID
		//Check for anonymous permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with 404
			logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "authError")
			return
		}
	}
	if !access.Access.HasAccess(interfaces.Read) {
		logging.WriteLog(logging.LogLevelInfo, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"User does not have permission to this page"})
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "authError")
		return
	}

	//Get page data, fill out crumbs
	err = FillTemplatePageData(PageID, &TemplateInput)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page data", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Note does not exist or form filled incorrectly", "moveError")
		return
	}

	//Parse page data
	parsedData, err := GetParsedPage(TemplateInput.PageData.Content)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to parse page contents", err.Error()})
		TemplateInput.HTMLMessage = template.HTML("Failed to parse page contents. Please check page contents for issues in the markdown.")
	} else {
		TemplateInput.PageContent = parsedData
	}

	//Send in template
	replyWithTemplate("page.html", TemplateInput, responseWriter, request)
}

//PageResourceRouter serves requests to /page/{pageID}/resources/{resource}
func PageResourceRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]
	resource := urlVariables["resource"]

	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with 404
		logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageResourceRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page resource", pageID, resource, err.Error()})
		http.Error(responseWriter, "", http.StatusNotFound)
		return
	}

	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	if TemplateInput.IsLoggedOn() {
		//Check user permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with 404
			logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
			http.Error(responseWriter, "", http.StatusInternalServerError)
			return
		}
	} else {
		access.User.DBID = interfaces.AnonymousUserID
		//Check for anonymous permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with 404
			logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
			http.Error(responseWriter, "", http.StatusInternalServerError)
			return
		}
	}
	if !access.Access.HasAccess(interfaces.Read) {
		http.Error(responseWriter, "", http.StatusNotFound)
		return
	}

	//Respond with file
	filePath := getPageResourcePath(PageID, resource)
	if _, err := os.Stat(filePath); err != nil {
		//If any error occurs, log it and respond with 404
		logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageResourceRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page resource", pageID, resource, err.Error()})
		http.Error(responseWriter, "", http.StatusNotFound)
		return
	}

	http.ServeFile(responseWriter, request, filePath)
}

//getPageResourceRootPath returns the filepath for a page's root
func getPageResourceRootPath(PageID uint64) string {
	return path.Join(config.Configuration.PageDirectory, strconv.FormatUint(PageID, 36))
}

//getPageResourcePath returns the filepath for a page's root
func getPageResourcePath(PageID uint64, Resource string) string {
	return path.Join(getPageResourceRootPath(PageID), Resource)
}
