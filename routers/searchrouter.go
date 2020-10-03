package routers

import (
	"html/template"
	"net/http"
	"z-notes/config"
	"z-notes/database"
	"z-notes/logging"
)

//SearchRouter serves requests to /search
func SearchRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)

	//Check if logged in
	if !TemplateInput.IsLoggedOn() {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelInfo, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"User not logged in for search"})
		redirectWithFlash(responseWriter, request, "/", "Access Denied, you must be logged in to search", "searchError")
		return
	}

	//Check if search was filled out
	if request.FormValue("Search") == "" {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelInfo, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"User did not provide a query"})
		redirectWithFlash(responseWriter, request, "/", "You did not provide a search query, try again", "searchError")
		return
	}

	//Perform Search
	TemplateInput.Title = "Search Results"

	//Grab result pages
	results, err := database.DBInterface.SearchPages(TemplateInput.UserInformation.DBID, request.FormValue("Search"), config.Configuration.MaxQueryResults, 0)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "pagerouter/PageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get search results", err.Error()})
		TemplateInput.HTMLMessage = template.HTML("Failed to get search results, internal error occured.")
	} else {
		TemplateInput.SearchResults = results
	}
	//Send in template
	replyWithTemplate("search.html", TemplateInput, responseWriter, request)
}
