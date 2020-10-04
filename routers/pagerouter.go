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

	//Grab page content
	pageData, err := database.DBInterface.GetPage(PageID)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page data", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Page requested could not be found", "pageError")
		return
	}

	//Parse content into HTML
	TemplateInput.Title = pageData.Name
	TemplateInput.PageData = pageData
	parsedData, err := GetParsedPage(pageData.Content)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to parse page contents", err.Error()})
		TemplateInput.HTMLMessage = template.HTML("Failed to parse page contents. Please check page contents for issues in the markdown.")
	} else {
		TemplateInput.PageContent = parsedData
	}

	//Grab child pages so that the menu may be constructed in template
	children, err := database.DBInterface.GetPageChildren(PageID)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get child pages", err.Error()})
		TemplateInput.HTMLMessage = template.HTML("Failed to get child pages, internal error occured.")
	} else {
		TemplateInput.ChildPages = children
	}
	//Grab ParentPageData for menu in template
	TemplateInput.ParentPageData = interfaces.Page{Name: "Library Root", OwnerID: TemplateInput.UserInformation.DBID}
	if pageData.PrevID != 0 {
		TemplateInput.ParentPageData, err = database.DBInterface.GetPage(pageData.PrevID)
		if err != nil {
			logging.WriteLog(logging.LogLevelError, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to parse get parent page from database", err.Error()})
		}
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
