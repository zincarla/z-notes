package routers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
	"z-notes/database"
	"z-notes/interfaces"
	"z-notes/logging"
)

//TokenGetRouter serves requests to /tokens
func TokenGetRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	TemplateInput.Title = "Token Management"
	var err error

	if !TemplateInput.IsLoggedOn() {
		logging.WriteLog(logging.LogLevelWarning, "tokenpage/TokenGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"User is not logged in"})
		redirectWithFlash(responseWriter, request, "/", "You must be logged in to manage tokens", "authError")
		return
	}
	FillLibraryWithRoot(&TemplateInput)

	//Load user's tokens
	TemplateInput.UserTokens, err = database.DBInterface.GetTokens(TemplateInput.UserInformation.DBID)
	if err != sql.ErrNoRows && err != nil {
		logging.WriteLog(logging.LogLevelWarning, "tokenpage/TokenGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Database failure in loading tokens", err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Failed to load tokens", "internalError")
		return
	}

	//Reply with token form
	replyWithTemplate("tokenpage.html", TemplateInput, responseWriter, request)
}

//TokenPagePostRouter serves requests to /token
func TokenPagePostRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	var err error

	if !TemplateInput.IsLoggedOn() {
		logging.WriteLog(logging.LogLevelWarning, "tokenpage/TokenGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"User is not logged in"})
		redirectWithFlash(responseWriter, request, "/", "You must be logged in to manage tokens", "authError")
		return
	}

	//Grab form action
	action := strings.ToLower(request.FormValue("action"))

	if action == "create" {
		newAPIToken := interfaces.APITokenInformation{OwnerID: TemplateInput.UserInformation.DBID}
		newAPIToken.Expires = (strings.ToLower(request.FormValue("expires")) == "checked")

		if newAPIToken.Expires {
			expireTimeString := request.FormValue("expireTime")
			//Re-Expand seconds if missing
			if strings.Count(expireTimeString, ":") == 1 {
				expireTimeString = expireTimeString + ":00"
			}
			newAPIToken.ExpirationTime, err = time.Parse("2006-01-02T15:04:05", expireTimeString)
			if err != nil {
				logging.WriteLog(logging.LogLevelError, "tokenspage/TokenPagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"failed to parse expiration time for token", expireTimeString})
				redirectWithFlash(responseWriter, request, "/tokens", "Failed to create token, expiration time could not be parsed", "tokenError")
				return
			}
		}

		tokenData, err := database.DBInterface.CreateToken(newAPIToken)
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "tokenpage/TokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to create token", err.Error()})
			redirectWithFlash(responseWriter, request, "/tokens", "Failed to create token, internal error", "tokenError")
			return
		}
		redirectWithFlash(responseWriter, request, "/tokens", "Token created: "+tokenData.FriendlyID, "tokenSuccess")
		return
	}
	if action == "delete" {
		targetTokenFriendlyID := request.FormValue("tokenid")

		tokenData, err := database.DBInterface.GetToken(targetTokenFriendlyID)
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "tokenpage/TokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to find token to delete", targetTokenFriendlyID, err.Error()})
			redirectWithFlash(responseWriter, request, "/tokens", "Failed to find token, internal error", "tokenError")
			return
		}

		//Verify logged in user owns the token
		if tokenData.OwnerID != TemplateInput.UserInformation.DBID {
			logging.WriteLog(logging.LogLevelWarning, "tokenpage/TokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Attempted to delete another users token", targetTokenFriendlyID, err.Error()})
			redirectWithFlash(responseWriter, request, "/tokens", "Failed to find token, internal error", "tokenError")
			return
		}

		err = database.DBInterface.RemoveToken(targetTokenFriendlyID)
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "tokenpage/TokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to delete token", targetTokenFriendlyID, err.Error()})
			redirectWithFlash(responseWriter, request, "/tokens", "Failed to delete token, internal error", "tokenError")
			return
		}
		redirectWithFlash(responseWriter, request, "/tokens", "Token deleted", "tokenSuccess")
		return
	}
	if action == "refresh" {
		targetTokenFriendlyID := request.FormValue("tokenid")

		tokenData, err := database.DBInterface.GetToken(targetTokenFriendlyID)
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "tokenpage/TokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to find token to refresh", targetTokenFriendlyID, err.Error()})
			redirectWithFlash(responseWriter, request, "/tokens", "Failed to find token, internal error", "tokenError")
			return
		}

		//Verify logged in user owns the token
		if tokenData.OwnerID != TemplateInput.UserInformation.DBID {
			logging.WriteLog(logging.LogLevelWarning, "tokenpage/TokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Attempted to refresh another users token", targetTokenFriendlyID, err.Error()})
			redirectWithFlash(responseWriter, request, "/tokens", "Failed to find token, internal error", "tokenError")
			return
		}

		tokenData.Expires = (strings.ToLower(request.FormValue("expires")) == "checked")
		if tokenData.Expires {
			expireTimeString := request.FormValue("expireTime")
			//Re-Expand seconds if missing
			if strings.Count(expireTimeString, ":") == 1 {
				expireTimeString = expireTimeString + ":00"
			}
			tokenData.ExpirationTime, err = time.Parse("2006-01-02T15:04:05", expireTimeString)
			if err != nil {
				logging.WriteLog(logging.LogLevelError, "tokenspage/TokenPagePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"failed to parse expiration time for token", expireTimeString})
				redirectWithFlash(responseWriter, request, "/tokens", "Failed to create token, expiration time could not be parsed", "tokenError")
				return
			}
		}

		tokenData, err = database.DBInterface.RefreshToken(tokenData)
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "tokenpage/TokenPostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to delete token", targetTokenFriendlyID, err.Error()})
			redirectWithFlash(responseWriter, request, "/tokens", "Failed to refresh token, internal error", "tokenError")
			return
		}

		redirectWithFlash(responseWriter, request, "/tokens", "Token refreshed: "+tokenData.FriendlyID, "tokenSuccess")
		return
	}
	redirectWithFlash(responseWriter, request, "/tokens", "Posted form action not recognized", "tokenError")
	return
}
