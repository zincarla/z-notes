package api

import (
	"encoding/json"
	"net"
	"net/http"
	"time"
	"z-notes/config"
	"z-notes/interfaces"
	"z-notes/logging"

	"github.com/gorilla/csrf"
)

//ErrorResponse used to marshal error text for JSON parsing
type ErrorResponse struct {
	Error string
}

//ThrottleErrorResponse used to reply error text and time in milliseconds
type ThrottleErrorResponse struct {
	ErrorResponse
	Timeout int64
}

//GenericResponse used to reply with a simple text-based result
type GenericResponse struct {
	//Result Generic result message
	Result string
	//Time it took server to process request
	RequestTime int64
	//Data is a generic interface for other response data to be plugged into
	Data interface{}
}

//APIData used to keep track of generic API data in a request
type APIData struct {
	Version string

	UserInformation interfaces.UserInformation

	//RequestStart is start time for a user request
	RequestStart time.Time
	//RequestTime is time it took to process a user request in MS
	RequestTime int64
}

//IsLoggedOn returns whether the user is to be treated as logged in
func (ti APIData) IsLoggedOn() bool {
	return ti.UserInformation.OIDCSubject != "" && ti.UserInformation.OIDCIssuer != ""
}

//ReplyWithJSON replies to a request with the specified interface to be marshaled to a JOSN object
func ReplyWithJSON(responseWriter http.ResponseWriter, request *http.Request, jsonObject interface{}, apiData APIData) {
	ReplyWithJSONStatus(responseWriter, request, jsonObject, apiData, http.StatusOK)
}

//ReplyWithJSONStatus replies to a request with the specified interface to be marshaled to a JOSN object and a custom status code
func ReplyWithJSONStatus(responseWriter http.ResponseWriter, request *http.Request, jsonObject interface{}, apiData APIData, statusCode int) {
	responseWriter.Header().Set("X-CSRF-Token", csrf.Token(request)) //Reply CSRF so client can make change requests

	finalResponse := GenericResponse{Result: "SUCCESS", Data: jsonObject}
	if statusCode != http.StatusOK {
		finalResponse.Result = "ERROR"
	}
	finalResponse.RequestTime = time.Now().Sub(apiData.RequestStart).Nanoseconds() / 1000000

	response, err := json.Marshal(finalResponse)
	if err != nil {
		logging.WriteLog(logging.LogLevelError, "apiroot/ReplyWithJSONStatus", apiData.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error generating JSON during reply", err.Error()})
		http.Error(responseWriter, "{\"Error\": \"Internal error generating response\", \"Result\": \"ERROR\"}", http.StatusInternalServerError)
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(statusCode)
	responseWriter.Write(response)
}

//ReplyWithJSONError replies to a request with an error response
func ReplyWithJSONError(responseWriter http.ResponseWriter, request *http.Request, errorText string, apiData APIData, statusCode int) {
	logging.WriteLog(logging.LogLevelVerbose, "apiroot/ReplyWithJSONError", apiData.UserInformation.GetCompositeID(), logging.ResultFailure, []string{errorText})
	ReplyWithJSONStatus(responseWriter, request, ErrorResponse{Error: errorText}, apiData, statusCode)
}

//ReplyWithLogonRequired replies to a request with an error response stating authentication required
func ReplyWithLogonRequired(responseWriter http.ResponseWriter, request *http.Request, apiData APIData) {
	//TODO: Update this header to be consistent with future chosen API auth method
	responseWriter.Header().Add("WWW-Authenticate", "Newauth realm=\"znotes-api\"") //IANA requires a WWW-Authenticate header with StatusUnauthorized,
	ReplyWithJSONError(responseWriter, request, "You must be logged in to use this API", apiData, http.StatusUnauthorized)
}

//GetAPIData returns an object for passing API data to functions
func GetAPIData(responseWriter http.ResponseWriter, request *http.Request) APIData {
	NewData := APIData{Version: config.ApplicationVersion,
		RequestStart: time.Now()}

	session, err := config.SessionStore.Get(request, config.SessionVariableName)
	if err == nil {
		//Load user session
		if oidcUserInfo, ok := session.Values["oidcuserinfo"].(interfaces.UserInformation); ok {
			if config.Configuration.OpenIDLogonExpireTime > 0 {
				if oidcUserInfo.OIDCIssueTime.Add(time.Second * time.Duration(config.Configuration.OpenIDLogonExpireTime)).After(time.Now()) {
					NewData.UserInformation = oidcUserInfo
				} //Else expired
			} else {
				NewData.UserInformation = oidcUserInfo
			}
		}
	}

	//TODO: Add API method of session validation. Current works for request from in-browser, but not so much from 3rd part integration
	//JWT?

	//Add IP to user info
	NewData.UserInformation.IP, _, err = net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		NewData.UserInformation.IP = request.RemoteAddr
	}

	return NewData
}
