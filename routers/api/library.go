package api

import (
	"net/http"
	"strconv"
	"z-notes/database"
	"z-notes/interfaces"
	"z-notes/logging"

	"github.com/gorilla/mux"
)

//NoteChildren returns a slice of notes under the requested note
type NoteChildren struct {
	Children    []interfaces.Page
	CurrentPage interfaces.Page
}

//NoteChildrenGetAPIRouter serves get requests to /api/notes/{id}/children
func NoteChildrenGetAPIRouter(responseWriter http.ResponseWriter, request *http.Request) {
	//Get apidata
	APIData := GetAPIData(responseWriter, request)
	//Validate Logon
	if !APIData.IsLoggedOn() {
		ReplyWithLogonRequired(responseWriter, request, APIData)
		return
	}
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]
	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with 404
		logging.WriteLog(logging.LogLevelWarning, "api/library/NoteChildrenGetAPIRouter", APIData.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", pageID, err.Error()})
		ReplyWithJSONError(responseWriter, request, "PageID not found", APIData, http.StatusNotFound)
		return
	}

	//Get current page, ensures it exists, and is needed for ParentID
	currentPage := interfaces.Page{Name: "Library Root", OwnerID: APIData.UserInformation.DBID}
	if PageID != 0 {
		currentPage, err = database.DBInterface.GetPage(PageID)
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "api/library/NoteChildrenGetAPIRouter", APIData.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get page", err.Error()})
			ReplyWithJSONError(responseWriter, request, "Internal error occured getting note", APIData, http.StatusInternalServerError)
			return
		}
	}

	//Get pages children
	var children []interfaces.Page
	if PageID == 0 {
		children, err = database.DBInterface.GetRootPages(APIData.UserInformation.DBID)
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "api/library/NoteChildrenGetAPIRouter", APIData.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get child pages", err.Error()})
			ReplyWithJSONError(responseWriter, request, "Internal error occured getting child notes", APIData, http.StatusInternalServerError)
			return
		}
	} else {
		children, err = database.DBInterface.GetPageChildren(PageID)
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "api/library/NoteChildrenGetAPIRouter", APIData.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get child pages", err.Error()})
			ReplyWithJSONError(responseWriter, request, "Internal error occured getting child notes", APIData, http.StatusInternalServerError)
			return
		}
	}
	ReplyWithJSON(responseWriter, request, NoteChildren{CurrentPage: currentPage, Children: children}, APIData)
}
