package routers

import (
	"net/http"
	"strconv"
	"z-notes/database"
	"z-notes/interfaces"
	"z-notes/logging"

	"github.com/gorilla/mux"
)

//SecurityPagePostRouter serves requests to /page/{pageID}/security/addToken
func SecurityPageTokenPostRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]
	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/SecurityPageTokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", urlVariables["pageID"], err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Note not found", "secError")
		return
	}
	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	if TemplateInput.IsLoggedOn() {
		//Check user permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "permissionpagetoken/SecurityPageTokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", urlVariables["pageID"], err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "secError")
			return
		}
	}
	if !access.Access.HasAccess(interfaces.Moderate) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "secError")
		return
	}

	//Verify user is set
	if request.FormValue("TokenID") == "" {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpagetoken/SecurityPageTokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"UserName not provided in security form", urlVariables["pageID"]})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "UserName cannot be blank", "secError")
		return
	}

	//Get token info
	tokenToEdit, err := database.DBInterface.GetToken(request.FormValue("TokenID"))
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpagetoken/SecurityPageTokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Token provided does not exist", urlVariables["pageID"], request.FormValue("TokenID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "Token not found", "secError")
		return
	}

	//Token exists. Now create an access object
	var newTokenAccess interfaces.PageAccessControl
	if request.FormValue("Deny") == "checked" {
		newTokenAccess = newTokenAccess | interfaces.Deny
	}
	if request.FormValue("Read") == "checked" {
		newTokenAccess = newTokenAccess | interfaces.Read
	}
	if request.FormValue("Write") == "checked" {
		newTokenAccess = newTokenAccess | interfaces.Write
	}
	if request.FormValue("Delete") == "checked" {
		newTokenAccess = newTokenAccess | interfaces.Delete
	}
	if request.FormValue("Audit") == "checked" {
		newTokenAccess = newTokenAccess | interfaces.Audit
	}
	if request.FormValue("Moderate") == "checked" {
		newTokenAccess = newTokenAccess | interfaces.Moderate
	}
	if request.FormValue("Inherits") == "checked" {
		newTokenAccess = newTokenAccess | interfaces.Inherits
	}

	//Now save permission to database
	if err = database.DBInterface.UpdateTokenPermission(interfaces.TokenPageAccess{Token: tokenToEdit, PageID: PageID, Access: newTokenAccess}); err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpagetoken/SecurityPageTokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to save permissions", urlVariables["pageID"], request.FormValue("TokenID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "Failed to save permissions", "secError")
		return
	}

	//Reply with redirect to saved page
	http.Redirect(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", http.StatusFound)
}

//SecurityPageDeletePostRouter serves requests to /page/{pageID}/security/deleteToken
func SecurityPageDeleteTokenPostRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]
	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpagetoken/SecurityPageDeleteTokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", urlVariables["pageID"], err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Note not found", "secError")
		return
	}
	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	if TemplateInput.IsLoggedOn() {
		//Check user permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "permissionpagetoken/SecurityPageDeleteTokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", urlVariables["pageID"], err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "secError")
			return
		}
	}
	if !access.Access.HasAccess(interfaces.Moderate) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "secError")
		return
	}

	//Verify id is set
	if request.FormValue("TokenAccessID") == "" {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpagetoken/SecurityPageDeleteTokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"TokenAccessID not provided in security form", urlVariables["pageID"]})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "TokenAccessID cannot be blank", "secError")
		return
	}

	//Parse
	AccessID, err := strconv.ParseUint(request.FormValue("TokenAccessID"), 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpagetoken/SecurityPageDeleteTokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing TokenAccessID", urlVariables["pageID"], request.FormValue("AccessID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "Access not found", "secError")
		return
	}

	//Verify access exists, and that it is for the requested page
	accessToDelete := interfaces.TokenPageAccess{ID: AccessID}
	accessToDelete, err = database.DBInterface.GetTokenPermission(accessToDelete)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpagetoken/SecurityPageDeleteTokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error getting access from database", urlVariables["pageID"], request.FormValue("AccessID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "Access not found", "secError")
		return
	}

	//Verify access is for this page, this is needed to ensure our permission check above is applicable
	if accessToDelete.PageID != PageID {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpagetoken/SecurityPageDeleteTokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"TokenAccessID is for access to another page", urlVariables["pageID"], request.FormValue("AccessID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "Access not found", "secError")
		return
	}

	//Now delete permission from database
	if err = database.DBInterface.RemoveTokenPermission(accessToDelete.ID); err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpagetoken/SecurityPageDeleteTokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to delete permissions", urlVariables["pageID"], request.FormValue("AccessID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "Failed to delete permissions, internal error", "secError")
		return
	}

	//Reply with redirect to saved page
	http.Redirect(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", http.StatusFound)
}
