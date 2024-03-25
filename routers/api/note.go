package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"z-notes/database"
	"z-notes/interfaces"
	"z-notes/logging"

	"github.com/gorilla/mux"
)

//NoteGetAPIRouter serves get requests to /api/notes/{id}
func NoteGetAPIRouter(responseWriter http.ResponseWriter, request *http.Request) {
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
		logging.WriteLog(logging.LogLevelWarning, "api/note/NoteGetAPIRouter", APIData.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", pageID, err.Error()})
		ReplyWithJSONError(responseWriter, request, "PageID not found", APIData, http.StatusNotFound)
		return
	}

	if PageID == 0 {
		//If any error occurs, log it and respond with 404
		logging.WriteLog(logging.LogLevelWarning, "api/note/NoteGetAPIRouter", APIData.GetCompositeID(), logging.ResultFailure, []string{"Invalid pageID", pageID})
		ReplyWithJSONError(responseWriter, request, "PageID not found", APIData, http.StatusNotFound)
		return
	}

	//Validate Permissions
	access, err := GetAPIDataAccess(APIData, PageID)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "api/note/NoteGetAPIRouter", APIData.GetCompositeID(), logging.ResultFailure, []string{"Failed to get page could not verify permissions", err.Error()})
		ReplyWithJSONError(responseWriter, request, "Internal error occured getting note", APIData, http.StatusInternalServerError)
		return
	}
	if !access.HasAccess(interfaces.Read) {
		logging.WriteLog(logging.LogLevelInfo, "api/note/NoteGetAPIRouter", APIData.GetCompositeID(), logging.ResultFailure, []string{"APIData does not have permission to this page"})
		ReplyWithJSONError(responseWriter, request, "Access Denied", APIData, http.StatusUnauthorized)
		return
	}

	currentPage, err := database.DBInterface.GetPage(PageID)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "api/note/NoteGetAPIRouter", APIData.GetCompositeID(), logging.ResultFailure, []string{"Failed to get page", err.Error()})
		ReplyWithJSONError(responseWriter, request, "Internal error occured getting note", APIData, http.StatusInternalServerError)
		return
	}

	ReplyWithJSON(responseWriter, request, notePostData{Name: currentPage.Name, Content: currentPage.Content}, APIData)
}

type notePostData struct {
	Name    string
	Content string
}

//NotePostAPIRouter serves get requests to /api/notes/{id}
func NotePostAPIRouter(responseWriter http.ResponseWriter, request *http.Request) {
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
		logging.WriteLog(logging.LogLevelWarning, "api/note/NoteGetAPIRouter", APIData.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", pageID, err.Error()})
		ReplyWithJSONError(responseWriter, request, "PageID not found", APIData, http.StatusNotFound)
		return
	}

	if PageID == 0 {
		//If any error occurs, log it and respond with 404
		logging.WriteLog(logging.LogLevelWarning, "api/note/NoteGetAPIRouter", APIData.GetCompositeID(), logging.ResultFailure, []string{"Invalid pageID", pageID})
		ReplyWithJSONError(responseWriter, request, "PageID not found", APIData, http.StatusNotFound)
		return
	}

	//Validate Permissions
	access, err := GetAPIDataAccess(APIData, PageID)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "api/note/NoteGetAPIRouter", APIData.GetCompositeID(), logging.ResultFailure, []string{"Failed to get page could not verify permissions", err.Error()})
		ReplyWithJSONError(responseWriter, request, "Internal error occured getting note", APIData, http.StatusInternalServerError)
		return
	}
	if !access.HasAccess(interfaces.Write) {
		logging.WriteLog(logging.LogLevelInfo, "api/note/NoteGetAPIRouter", APIData.GetCompositeID(), logging.ResultFailure, []string{"APIData does not have permission to this page"})
		ReplyWithJSONError(responseWriter, request, "Access Denied", APIData, http.StatusUnauthorized)
		return
	}

	currentPage, err := database.DBInterface.GetPage(PageID)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "api/note/NoteGetAPIRouter", APIData.GetCompositeID(), logging.ResultFailure, []string{"Failed to get page", err.Error()})
		ReplyWithJSONError(responseWriter, request, "Internal error occured getting note", APIData, http.StatusInternalServerError)
		return
	}

	//Parse user post JSON request
	decoder := json.NewDecoder(request.Body)
	var postedData notePostData
	if err := decoder.Decode(&postedData); err != nil {
		ReplyWithJSONError(responseWriter, request, "Failed to parse request data", APIData, http.StatusBadRequest)
		return
	}

	currentPage.Name = postedData.Name
	currentPage.Content = postedData.Content

	if err = database.DBInterface.UpdatePage(currentPage); err != nil {
		ReplyWithJSONError(responseWriter, request, "Failed to save posted data", APIData, http.StatusInternalServerError)
		return
	}

	ReplyWithJSON(responseWriter, request, "", APIData)
}
