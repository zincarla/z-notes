package routers

import (
	"html"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"z-notes/config"
	"z-notes/database"
	"z-notes/interfaces"
	"z-notes/logging"

	"github.com/gorilla/mux"
)

//RevisionRouter serves requests to /revisions
func RevisionRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]
	var SearchPage uint64

	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Page missing for revision history", "revisionError")
		return
	}
	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	if TemplateInput.IsLoggedOn() {
		//Check user permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "revisionError")
			return
		}
	} else {
		access.User.DBID = interfaces.AnonymousUserID
		//Check for anonymous permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "revisionError")
			return
		}
	}
	if !access.Access.HasAccess(interfaces.Read | interfaces.Audit) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "revisionError")
		return
	}

	if request.FormValue("searchPage") != "" {
		SearchPage, err = strconv.ParseUint(request.FormValue("searchPage"), 10, 64)
		if err != nil {
			//If any error occurs, log it and respond
			logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing searchPage", request.FormValue("searchPage"), err.Error()})
			TemplateInput.HTMLMessage = template.HTML("Failed to parse search page")
			SearchPage = 0
		}
	}

	if SearchPage > 0 {
		SearchPage -= 1
	}

	//Get page data, fill out crumbs
	err = FillTemplatePageData(PageID, &TemplateInput)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page data", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Note does not exist or form filled incorrectly", "moveError")
		return
	}

	TemplateInput.Title = TemplateInput.Title + " Revision History"

	//Grab revision pages
	results, maxCount, err := database.DBInterface.GetPageRevisions(PageID, config.Configuration.MaxQueryResults, SearchPage*config.Configuration.MaxQueryResults)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get revision results", err.Error()})
		TemplateInput.HTMLMessage = template.HTML("Failed to get revision results, internal error occured.")
	} else {
		for i := 0; i < len(results); i++ {
			results[i].Content = strings.ReplaceAll(results[i].Content, "\r\n\r\n", "\r\n")
			results[i].Content = strings.ReplaceAll(results[i].Content, "\n\n", "\n")
			if len(results[i].Content) > 300 {
				results[i].Content = results[i].Content[0:300] + "..."
			}
			results[i].Content = html.EscapeString(results[i].Content)
		}
		TemplateInput.SearchResults = results
		TemplateInput.PageMenu, err = GeneratePageMenu(int64(SearchPage*config.Configuration.MaxQueryResults), int64(config.Configuration.MaxQueryResults), int64(maxCount), "/page/"+strconv.FormatUint(PageID, 10)+"/revisions")
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to generate page menu", err.Error()})
		}
	}

	//Reply with revision form
	replyWithTemplate("revisionpage.html", TemplateInput, responseWriter, request)
}

//RevisionRouter serves requests to /page/{pageID}/revision/{revisionID}
func RevisionViewRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]
	revisionID := urlVariables["revisionID"]

	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionViewRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Page missing for revision history", "revisionError")
		return
	}
	//Convert RevisionID
	RevisionID, err := strconv.ParseUint(revisionID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionViewRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing revisionID", revisionID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Revision ID not provided or incorrect", "revisionError")
		return
	}
	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	if TemplateInput.IsLoggedOn() {
		//Check user permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionViewRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "revisionError")
			return
		}
	} else {
		access.User.DBID = interfaces.AnonymousUserID
		//Check for anonymous permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionViewRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "revisionError")
			return
		}
	}
	if !access.Access.HasAccess(interfaces.Read | interfaces.Audit) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "revisionError")
		return
	}

	//Get page data, fill out crumbs
	err = FillTemplatePageData(PageID, &TemplateInput)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionViewRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page data", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Note does not exist or form filled incorrectly", "revisionError")
		return
	}

	//Grab revision page
	revisionPage, err := database.DBInterface.GetPageRevision(PageID, RevisionID)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionViewRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get revision", err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Failed to load revision", "revisionError")
		return
	}

	TemplateInput.Title = TemplateInput.Title + " [Revision: " + revisionPage.Name + "]"
	TemplateInput.PageData.RevisionID = RevisionID
	TemplateInput.PageData.Name = revisionPage.Name
	TemplateInput.PageData.Content = revisionPage.Content
	TemplateInput.PageData.RevisionTime = revisionPage.RevisionTime

	//Parse page data
	parsedData, err := GetParsedPage(TemplateInput.PageData.Content)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "revisionrouter/RevisionViewRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to parse page revision contents", err.Error()})
		TemplateInput.HTMLMessage = template.HTML("Failed to parse page contents. Please check page contents for issues in the markdown.")
	} else {
		TemplateInput.PageContent = parsedData
	}

	//Reply with revision form
	replyWithTemplate("page.html", TemplateInput, responseWriter, request)
}
