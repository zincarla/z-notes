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

//MovePageGetRouter serves requests to /page/id/move?ParentPageID=* (This does not actually move the page, this shows the form to move the page. User can keep navigating library)
func MovePageGetRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]

	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "movepage/MovePageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Page not found", "moveError")
		return
	}

	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	if TemplateInput.IsLoggedOn() {
		//Check user permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "movepage/MovePageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "moveError")
			return
		}
	} else {
		redirectWithFlash(responseWriter, request, "/", "You must log on first", "moveError")
		return
	}
	if !access.Access.HasAccess(interfaces.Read | interfaces.Write | interfaces.Delete) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "moveError")
		return
	}

	//Set parentPageID
	var parentPageID uint64
	if request.FormValue("ParentPageID") != "" {
		parentPageID, err = strconv.ParseUint(request.FormValue("ParentPageID"), 10, 64)
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "movepage/MovePageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to parse parent page id", request.FormValue("ParentPageID"), err.Error()})
			TemplateInput.HTMLMessage += template.HTML("Failed to parse parentID. Redirecting to library root. ")
		}
	}

	//Grab parent page data
	newParentPage := interfaces.Page{OwnerID: TemplateInput.UserInformation.DBID, Name: "Library Root"} //Defaults to 0 ID
	if parentPageID != 0 {
		newParentPage, err = database.DBInterface.GetPage(parentPageID)
		if err != nil {
			logging.WriteLog(logging.LogLevelError, "movepage/MovePagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting parent page data", strconv.FormatUint(parentPageID, 10), err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Page not found", "moveError")
			return
		}
	}
	TemplateInput.MovingParentPageData = newParentPage

	//Get page data, fill out crumbs, children
	err = FillTemplatePageData(PageID, &TemplateInput)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "movepage/MovePageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page data", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Note does not exist or form filled incorrectly", "moveError")
		return
	}

	//For navigation, get children of parentPageID we will reuse searchResults
	if parentPageID == 0 {
		TemplateInput.SearchResults, err = database.DBInterface.GetRootPages(TemplateInput.UserInformation.DBID)
	} else {
		TemplateInput.SearchResults, err = database.DBInterface.GetPageChildren(parentPageID)
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "movepage/MovePageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get child of parent pages", err.Error()})
			TemplateInput.HTMLMessage = template.HTML("Failed to get menu pages, internal error occured.")
		}
	}

	//Reply with edit form
	replyWithTemplate("movepage.html", TemplateInput, responseWriter, request)
}

//MovePagePostRouter serves requests to /page/id/move
func MovePagePostRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]
	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "movepage/MovePagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Form filled incorrectly", "moveError")
		return
	}
	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	if TemplateInput.IsLoggedOn() {
		//Check user permissions
		access, err = database.DBInterface.GetEffectivePermission(access)
		if err != nil {
			//If any error occurs, log it and respond with redirect
			logging.WriteLog(logging.LogLevelWarning, "movepage/MovePageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "moveError")
			return
		}
	} else {
		redirectWithFlash(responseWriter, request, "/", "You must log on first", "moveError")
		return
	}
	if !access.Access.HasAccess(interfaces.Write | interfaces.Delete) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "moveError")
		return
	}

	//Set parentPageID
	var parentPageID uint64
	if request.FormValue("ParentPageID") != "" {
		parentPageID, err = strconv.ParseUint(request.FormValue("ParentPageID"), 10, 64)
		if err != nil {
			redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/move", "Intended parent page not found", "moveError")
			return
		}
	}

	//Check permissions on intended parent
	if parentPageID != 0 {
		//We only check if we are not trying to move page to the library root, user always has access to their own root
		access = interfaces.UserPageAccess{PageID: parentPageID, User: TemplateInput.UserInformation}
		if TemplateInput.IsLoggedOn() {
			//Check user permissions
			access, err = database.DBInterface.GetEffectivePermission(access)
			if err != nil {
				//If any error occurs, log it and respond with redirect
				logging.WriteLog(logging.LogLevelWarning, "movepage/MovePageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", pageID, err.Error()})
				redirectWithFlash(responseWriter, request, "/", "Access Denied", "moveError")
				return
			}
		} else {
			redirectWithFlash(responseWriter, request, "/", "You must log on first", "moveError")
			return
		}
		if !access.Access.HasAccess(interfaces.Write) {
			redirectWithFlash(responseWriter, request, "/", "Access Denied", "moveError")
			return
		}
	}

	//Get both pages data
	movingPageData, err := database.DBInterface.GetPage(PageID)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "movepage/MovePagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page data", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Page not found", "moveError")
		return
	}

	newParentPage := interfaces.Page{OwnerID: movingPageData.OwnerID} //Defaults to 0 ID
	if parentPageID != 0 {
		newParentPage, err = database.DBInterface.GetPage(parentPageID)
		if err != nil {
			logging.WriteLog(logging.LogLevelError, "movepage/MovePagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page data", strconv.FormatUint(parentPageID, 10), err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Page not found", "moveError")
			return
		}
	}

	//Verify both pages are owned by same user
	if newParentPage.OwnerID != movingPageData.OwnerID {
		logging.WriteLog(logging.LogLevelWarning, "movepage/MovePagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured moving page. The page cannot be moved outside the original user's library.", pageID, strconv.FormatUint(parentPageID, 10)})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/view", "Notes cannot be moved out of their owner's library", "moveError")
		return
	}

	//Next, check we are not about to move page into itself by comaring it's IDs directly
	if movingPageData.ID == newParentPage.ID {
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/view", "You cannot move a page into itself.", "moveError")
		return
	}
	//Then check the parent pages, to ensure we do not create a loop in the tree
	if newParentPage.ID != 0 { //No need to check if moving to root
		pagePath, err := database.DBInterface.GetPagePath(newParentPage.ID, false)
		if err != nil {
			logging.WriteLog(logging.LogLevelError, "movepage/MovePagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured moving page. The page's path could not be retrived.", pageID, strconv.FormatUint(parentPageID, 10), err.Error()})
			redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/view", "Page could not be moved, internal error", "moveError")
			return
		}
		for _, pageInPath := range pagePath {
			if pageInPath.ID == movingPageData.ID {
				redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/view", "You cannot move a page into itself.", "moveError")
				return
			}
		}
	}

	//Finally we can move the note
	movingPageData.PrevID = newParentPage.ID

	//Save the page update
	err = database.DBInterface.UpdatePage(movingPageData)
	if err != nil {
		logging.WriteLog(logging.LogLevelError, "movepage/MovePagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured setting page data", pageID, err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Internal error occurred", "moveError")
		return
	}

	//Reply with redirect to saved page
	http.Redirect(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/view", http.StatusFound)
}
