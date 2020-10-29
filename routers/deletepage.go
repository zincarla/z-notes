package routers

import (
	"errors"
	"net/http"
	"strconv"
	"z-notes/database"
	"z-notes/interfaces"
	"z-notes/logging"

	"github.com/gorilla/mux"
)

//DeletePagePostRouter serves requests to /delete
func DeletePagePostRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]
	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "editpage/DeletePagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Form filled incorrectly", "deleteError")
		return
	}
	//Check if logged in
	if !TemplateInput.IsLoggedOn() {
		redirectWithFlash(responseWriter, request, "/", "You must log on first", "deleteError")
		return
	}

	//Verify delete permission on tree
	err = VerifyChildPermission(TemplateInput.UserInformation.DBID, PageID, interfaces.Delete)
	if err != nil {
		logging.WriteLog(logging.LogLevelError, "editpage/DeletePagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured deleting page data", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Access denied on the note, or one of it's children", "deleteError")
		return
	}

	//Delete the page
	err = database.DBInterface.RemovePage(PageID)
	if err != nil {
		logging.WriteLog(logging.LogLevelError, "editpage/DeletePagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured deleting page data", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Internal error occurred", "deleteError")
		return
	}

	//Reply with redirect and message
	redirectWithFlash(responseWriter, request, "/", "Note Deleted", "deleteSuccess")
}

//VerifyChildPermission returns nil if user has the specified permission on the specified page and all children, otherwise an error
func VerifyChildPermission(userID uint64, rootPageID uint64, requiredPermission interfaces.PageAccessControl) error {
	//Grab page data
	pageData, err := database.DBInterface.GetPage(rootPageID)
	if err != nil {
		return err
	}
	if pageData.OwnerID == userID {
		return nil //Short-circuit outta here if the owner is the user being verified
	}

	currentWave := []uint64{rootPageID}
	nextWave := []uint64{}
	for len(currentWave) > 0 {

		for _, pageID := range currentWave {
			//Verify access for this page
			access := interfaces.UserPageAccess{PageID: pageID, User: interfaces.UserInformation{DBID: userID}}
			access, err := database.DBInterface.GetEffectivePermission(access)
			if err != nil {
				return err
			}
			if !access.Access.HasAccess(requiredPermission) {
				return errors.New("Access denied")
			}
			//userID has access to this page, so grab children and add to next wave
			children, err := database.DBInterface.GetPageChildren(pageID)
			if err != nil {
				return err
			}
			for _, child := range children {
				nextWave = append(nextWave, child.ID)
			}
		}

		//Reset for next loop
		currentWave = nextWave
		nextWave = []uint64{}
	}
	return nil
}
