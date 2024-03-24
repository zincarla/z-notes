package routers

import (
	"net/http"
	"strconv"
	"z-notes/database"
	"z-notes/interfaces"
	"z-notes/logging"

	"github.com/gorilla/mux"
)

//EditPageGetRouter serves requests to /page/{pageID}/edit
func EditPageGetRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]
	var RevisionID uint64

	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "editpage/EditPageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Edit form filled incorrectly", "editError")
		return
	}
	if request.FormValue("revisionID") != "" {
		RevisionID, err = strconv.ParseUint(request.FormValue("revisionID"), 10, 64)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "editpage/EditPageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing revisionID", request.FormValue("revisionID"), err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Edit form filled incorrectly", "editError")
			return
		}
	}

	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	if TemplateInput.IsLoggedOn() {
		//Check user permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "editpage/EditPageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "editError")
			return
		}
	} else {
		access.User.DBID = interfaces.AnonymousUserID
		//Check for anonymous permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "editpage/EditPageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "editError")
			return
		}
	}
	if !access.Access.HasAccess(interfaces.Read | interfaces.Write) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "editError")
		return
	}
	if RevisionID != 0 && !access.Access.HasAccess(interfaces.Audit) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "editError")
		return
	}

	//Get page data, fill out crumbs
	err = FillTemplatePageData(PageID, &TemplateInput)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "editpage/EditPageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page data", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Note does not exist or form filled incorrectly", "editError")
		return
	}

	if RevisionID != 0 {
		//Grab revision page
		revisionPage, err := database.DBInterface.GetPageRevision(PageID, RevisionID)
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "editpage/EditPageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get revision", err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Failed to load revision", "editError")
			return
		}

		TemplateInput.Title = TemplateInput.Title + " [Revision: " + revisionPage.Name + "]"
		TemplateInput.PageData.RevisionID = RevisionID
		TemplateInput.PageData.Name = revisionPage.Name
		TemplateInput.PageData.Content = revisionPage.Content
		TemplateInput.PageData.RevisionTime = revisionPage.RevisionTime
	}

	//Reply with edit form
	replyWithTemplate("editpage.html", TemplateInput, responseWriter, request)
}

//EditPagePostRouter serves requests to /page/{pageID}/edit
func EditPagePostRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]
	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "editpage/EditPagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Form filled incorrectly", "editError")
		return
	}
	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	if TemplateInput.IsLoggedOn() {
		//Check user permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "editpage/EditPagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", request.FormValue("PageID"), err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "editError")
			return
		}
	} else {
		access.User.DBID = interfaces.AnonymousUserID
		//Check for anonymous permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "editpage/EditPagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", request.FormValue("PageID"), err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "editError")
			return
		}
	}
	if !access.Access.HasAccess(interfaces.Write) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "editError")
		return
	}

	//Verify name is set
	if request.FormValue("PageName") == "" {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "editpage/EditPagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Page name cannot be blank", "editError")
		return
	}

	//Grab old page data
	pageData, err := database.DBInterface.GetPage(PageID)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "editpage/EditPagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page data", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Form filled incorrectly, or note does not exist", "editError")
		return
	}

	pageData.Name = request.FormValue("PageName")
	pageData.Content = request.FormValue("PageContent")

	//Save the page updates
	err = database.DBInterface.UpdatePage(pageData)
	if err != nil {
		logging.WriteLog(logging.LogLevelError, "editpage/EditPagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page data", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Internal error occurred", "editError")
		return
	}

	//Reply with redirect to saved page
	http.Redirect(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/view", http.StatusFound)
}
