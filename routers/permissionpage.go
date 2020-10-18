package routers

import (
	"html/template"
	"net/http"
	"strconv"
	"z-notes/database"
	"z-notes/interfaces"
	"z-notes/logging"

	"github.com/gorilla/mux"
)

//SecurityPageGetRouter serves requests to /page/{pageID}/security
func SecurityPageGetRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]

	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/SecurityPageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Page not found", "secError")
		return
	}
	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	if TemplateInput.IsLoggedOn() {
		//Check user permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "permissionpage/SecurityPageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "secError")
			return
		}
	}
	if !access.Access.HasAccess(interfaces.Moderate) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "secError")
		return
	}

	//Get page data/crumbs
	err = FillTemplatePageData(PageID, &TemplateInput)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/SecurityPageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page data", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Note does not exist or form filled incorrectly", "moveError")
		return
	}

	//Get permissions
	permissions, err := database.DBInterface.GetPermissions(PageID)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/SecurityPageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get page permissions", err.Error()})
		TemplateInput.HTMLMessage = template.HTML("Failed to get page permissions, internal error occured.")
	} else {
		//By default, permissions only has the DBID of the users, for UI, we need the usernames too
		for index := range permissions {
			completedUser, err := database.DBInterface.GetUser(permissions[index].User)
			if err != nil {
				TemplateInput.HTMLMessage = template.HTML("Failed to get all user info.")
				logging.WriteLog(logging.LogLevelWarning, "permissionpage/SecurityPageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get users for page permissions", err.Error()})
				permissions[index].User.Name = "FailedUser#" + strconv.FormatUint(permissions[index].User.DBID, 10)
			} else {
				permissions[index].User = completedUser
			}
		}
		TemplateInput.PagePermissions = permissions
	}

	//Reply with edit form
	replyWithTemplate("editsecurity.html", TemplateInput, responseWriter, request)
}

//SecurityPagePostRouter serves requests to /page/{pageID}/security/add
func SecurityPagePostRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]
	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", urlVariables["pageID"], err.Error()})
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
			logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", urlVariables["pageID"], err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "secError")
			return
		}
	}
	if !access.Access.HasAccess(interfaces.Moderate) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "secError")
		return
	}

	//Verify user is set
	if request.FormValue("UserName") == "" {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"UserName not provided in security form", urlVariables["pageID"], err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "UserName cannot be blank", "secError")
		return
	}

	//Verify user exists
	userToEdit := interfaces.UserInformation{}
	if err = userToEdit.SetName(request.FormValue("UserName")); err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"UserName provided does not exist", urlVariables["pageID"], request.FormValue("UserName"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "User not found", "secError")
		return
	}
	if userToEdit.DBID == 0 {
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"UserName provided does not exist, id 0", urlVariables["pageID"], request.FormValue("UserName")})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "User not found", "secError")
		return
	}

	userToEdit, err = database.DBInterface.GetUser(userToEdit)
	if err = userToEdit.SetName(request.FormValue("UserName")); err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"UserName provided does not exist", urlVariables["pageID"], request.FormValue("UserName"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "User not found", "secError")
		return
	}

	//User exists. Now create an access object
	var newUserAccess interfaces.PageAccessControl
	if request.FormValue("Deny") == "checked" {
		newUserAccess = newUserAccess | interfaces.Deny
	}
	if request.FormValue("Read") == "checked" {
		newUserAccess = newUserAccess | interfaces.Read
	}
	if request.FormValue("Write") == "checked" {
		newUserAccess = newUserAccess | interfaces.Write
	}
	if request.FormValue("Delete") == "checked" {
		newUserAccess = newUserAccess | interfaces.Delete
	}
	if request.FormValue("Audit") == "checked" {
		newUserAccess = newUserAccess | interfaces.Audit
	}
	if request.FormValue("Moderate") == "checked" {
		newUserAccess = newUserAccess | interfaces.Moderate
	}
	if request.FormValue("Inherits") == "checked" {
		newUserAccess = newUserAccess | interfaces.Inherits
	}

	//Now save permission to database
	if err = database.DBInterface.UpdatePermission(interfaces.UserPageAccess{User: userToEdit, PageID: PageID, Access: newUserAccess}); err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to save permissions", urlVariables["pageID"], request.FormValue("UserName"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "Failed to save permissions", "secError")
		return
	}

	//Reply with redirect to saved page
	http.Redirect(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", http.StatusFound)
}

//SecurityPageDeletePostRouter serves requests to /page/{pageID}/security/delete
func SecurityPageDeletePostRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]
	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", urlVariables["pageID"], err.Error()})
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
			logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", urlVariables["pageID"], err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "secError")
			return
		}
	}
	if !access.Access.HasAccess(interfaces.Moderate) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "secError")
		return
	}

	//Verify id is set
	if request.FormValue("AccessID") == "" {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"AccessID not provided in security form", urlVariables["pageID"], err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "AccessID cannot be blank", "secError")
		return
	}

	//Parse
	AccessID, err := strconv.ParseUint(request.FormValue("AccessID"), 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing AccessID", urlVariables["pageID"], request.FormValue("AccessID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "Access not found", "secError")
		return
	}

	//Verify access exists, and that it is for the requested page
	accessToDelete := interfaces.UserPageAccess{ID: AccessID}
	accessToDelete, err = database.DBInterface.GetPermission(accessToDelete)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error getting access from database", urlVariables["pageID"], request.FormValue("AccessID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "Access not found", "secError")
		return
	}

	//Verify access is for this page, this is needed to ensure our permission check above is applicable
	if accessToDelete.PageID != PageID {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"AccessID is for access to another page", urlVariables["pageID"], request.FormValue("AccessID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "Access not found", "secError")
		return
	}

	//Now delete permission from database
	if err = database.DBInterface.RemovePermission(accessToDelete.ID); err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "permissionpage/permissionpagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to delete permissions", urlVariables["pageID"], request.FormValue("AccessID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", "Failed to delete permissions, internal error", "secError")
		return
	}

	//Reply with redirect to saved page
	http.Redirect(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/security", http.StatusFound)
}
