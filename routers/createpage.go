package routers

import (
	"net/http"
	"strconv"
	"z-notes/database"
	"z-notes/interfaces"
	"z-notes/logging"
)

//CreatePageRouter serves requests to /createpage
func CreatePageRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	if !TemplateInput.IsLoggedOn() {
		redirectWithFlash(responseWriter, request, "/", "You must be logged in to perform this action", "authError")
		return
	}

	//First, verify name
	if request.FormValue("NoteName") == "" {
		//Set flash
		redirectWithFlash(responseWriter, request, "/", "A name is required when creating a note", "createError")
		return
	}

	//Cache OwnerID, this will be changed if page does not belong to current user
	OwnerID := TemplateInput.UserInformation.DBID

	//Second, verify ParentID
	ParentID, err := strconv.ParseUint(request.FormValue("ParentID"), 10, 64)
	if err != nil {
		//Set flash
		redirectWithFlash(responseWriter, request, "/", "Failed to parse note's parent ID", "createError")
		return
	}
	if ParentID != 0 { //If we are not adding a root page, then we need to check permissions on the page (Could be creating under a different profile for example)
		PageData, err := database.DBInterface.GetPage(ParentID)
		if err != nil {
			//Set flash
			logging.WriteLog(logging.LogLevelError, "createpage/CreatePageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get parent page for new note", err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Failed to get parent note", "createError")
			return
		}
		//Verify permissions
		userPermissions, err := database.DBInterface.GetEffectivePermission(interfaces.UserPageAccess{User: TemplateInput.UserInformation, PageID: PageData.ID})
		if err != nil {
			//Set flash
			logging.WriteLog(logging.LogLevelError, "createpage/CreatePageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get permissions", err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Failed to validate permissions", "createError")
			return
		}

		if !userPermissions.Access.HasAccess(interfaces.Write) {
			//Set flash
			logging.WriteLog(logging.LogLevelWarning, "createpage/CreatePageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"User does not have access to do this", err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "createError")
			return
		}

		OwnerID = PageData.OwnerID
	}

	//Permissions validated, information needed good, create note
	pageID, err := database.DBInterface.CreatePage(interfaces.Page{Name: request.FormValue("NoteName"), PrevID: ParentID, OwnerID: OwnerID, Content: "## " + request.FormValue("NoteName") + "\r\n\r\nWelcome to your new note!\r\n"})
	if err != nil {
		logging.WriteLog(logging.LogLevelError, "createpage/CreatePageRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to create user's note", err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Internal error occured creating note", "createError")
		return
	}

	//Redirect user to the newly made page
	http.Redirect(responseWriter, request, "/page/"+strconv.FormatUint(pageID, 10)+"/edit", http.StatusFound)
}
