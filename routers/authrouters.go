package routers

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/gob"
	"net/http"
	"time"
	"z-notes/config"
	"z-notes/database"
	"z-notes/interfaces"
	"z-notes/logging"

	"github.com/coreos/go-oidc"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/oauth2"
)

//TODO: Verify Context setting, is background good? maybe should add timeout?
//TODO: Verify cookie state expiration time, is 5 minutes good?

var oidcProvider *oidc.Provider
var oidcConfig oauth2.Config
var oidcVerifier *oidc.IDTokenVerifier

//InitOAuth should be called after loading and verifying application settings. Initiializes OAuth handlers
func InitOAuth() error {
	//Prepare gob
	gob.Register(interfaces.UserInformation{})
	//Load OIDC Config
	provider, err := oidc.NewProvider(context.Background(), config.Configuration.OpenIDEndpointURL)
	if err != nil {
		logging.WriteLog(logging.LogLevelCritical, "authrouters/InitOAuth", "", logging.ResultFailure, []string{"OpenIDC provider failed", err.Error()})
		return err
	}
	oidcProvider = provider
	//Create oidc config
	oidcConfig = oauth2.Config{
		ClientID:     config.Configuration.OpenIDClientID,
		ClientSecret: config.Configuration.OpenIDClientSecret,
		RedirectURL:  config.Configuration.OpenIDCallbackURL,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}
	oidcVerifier = oidcProvider.Verifier(&oidc.Config{ClientID: config.Configuration.OpenIDClientID})
	return nil
}

//AuthRouter serves requests to /openidc/logon
func AuthRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	//Set state cookie
	stateToken, err := uuid.NewV4()
	if err != nil {
		logging.WriteLog(logging.LogLevelError, "authrouters/AuthRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured setting getting new uuid for state cookie", err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Authentication failed", "authError")
		return
	}
	expiration := time.Now().Add(5 * time.Minute) //TODO: What is appropriate time? Minutes?
	state := base64.URLEncoding.EncodeToString(stateToken.Bytes())
	//Set data to session
	session, err := config.SessionStore.Get(request, config.SessionVariableName)
	session.Values["oauthstate"] = state
	session.Values["oauthexpiration"] = expiration.Format("2006-01-02 15:04:05.999999999 -0700 MST")
	err = session.Save(request, responseWriter) //Save session data
	if err != nil {
		logging.WriteLog(logging.LogLevelError, "authrouters/AuthRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to save session cookie", err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Authentication failed", "authError")
		return
	}

	//Redirect user to id provider
	http.Redirect(responseWriter, request, oidcConfig.AuthCodeURL(state), http.StatusFound)
}

//AuthLogoutRouter serves requests to /openidc/logout
func AuthLogoutRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)

	//Set data to session
	session, err := config.SessionStore.Get(request, config.SessionVariableName)
	session.Values["oidcuserinfo"] = ""
	session.Values["oauthstate"] = ""
	session.Values["oauthexpiration"] = ""
	err = session.Save(request, responseWriter) //Save session data
	if err != nil {
		logging.WriteLog(logging.LogLevelError, "authrouters/AuthLogoutRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to save session cookie", err.Error()})
	}

	//Redirect user to id provider
	http.Redirect(responseWriter, request, "/", http.StatusFound)
}

//AuthCallback serves responses to /openidc/callback
func AuthCallback(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)

	//Verify Cookie state
	//Load data from session
	session, err := config.SessionStore.Get(request, config.SessionVariableName)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "authrouters/AuthCallback", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to load session cookie", err.Error()})
	}
	stateString, ok := session.Values["oauthstate"].(string)
	if !ok || stateString == "" {
		logging.WriteLog(logging.LogLevelWarning, "authrouters/AuthCallback", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"OAuth state cookie could not parse state", stateString})
		redirectWithFlash(responseWriter, request, "/", "Internal error occured during authentication", "authError")
		return
	}
	if request.FormValue("state") != stateString {
		logging.WriteLog(logging.LogLevelWarning, "authrouters/AuthCallback", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"OAuth state cookie does not match callback state", stateString, request.FormValue("state")})
		redirectWithFlash(responseWriter, request, "/", "Internal error occured during authentication", "authError")
		return
	}
	expirationTimeString, ok := session.Values["oauthexpiration"].(string)
	if !ok {
		logging.WriteLog(logging.LogLevelWarning, "authrouters/AuthCallback", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"OAuth state cookie failed to parse expiration time"})
		redirectWithFlash(responseWriter, request, "/", "Internal error occured during authentication", "authError")
		return
	}
	expirationTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", expirationTimeString)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "authrouters/AuthCallback", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to parse expiration into time", expirationTimeString})
		redirectWithFlash(responseWriter, request, "/", "Internal error occured during authentication", "authError")
		return
	}
	if time.Now().After(expirationTime) {
		logging.WriteLog(logging.LogLevelWarning, "authrouters/AuthCallback", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"OAuth state cookie expired"})
		redirectWithFlash(responseWriter, request, "/", "Internal error occured during authentication", "authError")
		return
	}

	//Grab user data from oidc
	oauth2Token, err := oidcConfig.Exchange(context.Background(), request.FormValue("code"))
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "authrouters/AuthCallback", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed callback verification", err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Internal error occured during authentication", "authError")
		return
	}

	//Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		logging.WriteLog(logging.LogLevelWarning, "authrouters/AuthCallback", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed callback"})
		redirectWithFlash(responseWriter, request, "/", "Internal error occured during authentication", "authError")
		return
	}

	//Parse and verify ID Token payload.
	idToken, err := oidcVerifier.Verify(context.Background(), rawIDToken)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "authrouters/AuthCallback", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed callback verification", err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Internal error occured during authentication", "authError")
		return
	}

	//Extract custom claims
	var claims struct {
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
		Username string `json:"preferred_username"`
	}
	if err := idToken.Claims(&claims); err != nil {
		logging.WriteLog(logging.LogLevelWarning, "authrouters/AuthCallback", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed callback claims", err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Internal error occured during authentication", "authError")
		return
	}

	//Check if user exists
	authedUser := interfaces.UserInformation{Name: claims.Username, EMail: claims.Email, EMailVerified: claims.Verified, OIDCSubject: idToken.Subject, OIDCIssuer: idToken.Issuer, OIDCIssueTime: idToken.IssuedAt, IP: TemplateInput.UserInformation.IP}
	databaseUser, err := database.DBInterface.GetUser(authedUser)
	if err != nil && err == sql.ErrNoRows {
		//User does not exist
		if config.Configuration.AllowAccountCreation {
			if !authedUser.EMailVerified {
				//Reply with error
				logging.WriteLog(logging.LogLevelWarning, "authrouters/AuthCallback", authedUser.GetCompositeID(), logging.ResultFailure, []string{"Account create failed, user email not verified with OIDC provider"})
				redirectWithFlash(responseWriter, request, "/", "Failed to create account, please verify your email with your authentication provider", "authError")
				return
			}
			if len(authedUser.Name) > 255 {
				authedUser.Name = authedUser.Name[:255]
			}
			if _, err := database.DBInterface.CreateUser(authedUser); err != nil {
				//Reply with error
				logging.WriteLog(logging.LogLevelError, "authrouters/AuthCallback", authedUser.GetCompositeID(), logging.ResultFailure, []string{"Account create failed", err.Error()})
				redirectWithFlash(responseWriter, request, "/", "Failed to create account due to an internal error", "authError")
				return
			}
			databaseUser, err = database.DBInterface.GetUser(authedUser)
			if err != nil {
				//Failed to get user after adding, reply with error
				logging.WriteLog(logging.LogLevelError, "authrouters/AuthCallback", authedUser.GetCompositeID(), logging.ResultFailure, []string{"Account load after create failed", err.Error()})
				redirectWithFlash(responseWriter, request, "/", "Failed to create account due to an internal error", "authError")
				return
			}
		} else {
			//Reply with error
			logging.WriteLog(logging.LogLevelInfo, "authrouters/AuthCallback", authedUser.GetCompositeID(), logging.ResultFailure, []string{"Account not found, and creation disabled", err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Account not found and account creation is disabled on this server", "authError")
			return
		}
	} else if err != nil {
		//Failed to get user? Database connection error?
		logging.WriteLog(logging.LogLevelError, "authrouters/AuthCallback", authedUser.GetCompositeID(), logging.ResultFailure, []string{"Account check failed with database error", err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Internal error occured retrieving account info.", "authError")
		return
	}

	//Account should exists at this point, check if it needs updating by comparing authedUser and databaseUser
	authedUser.DBID = databaseUser.DBID
	logging.WriteLog(logging.LogLevelDebug, "authrouters/AuthCallback", authedUser.GetCompositeID(), logging.ResultInfo, []string{authedUser.Name, databaseUser.Name})
	if (authedUser.Name != databaseUser.Name && authedUser.Name != "") || (authedUser.EMail != databaseUser.EMail && authedUser.EMail != "") {
		logging.WriteLog(logging.LogLevelDebug, "authrouters/AuthCallback", authedUser.GetCompositeID(), logging.ResultInfo, []string{"Name/email update"})
		if len(authedUser.Name) > 255 {
			authedUser.Name = authedUser.Name[:255]
		}
		if err := database.DBInterface.UpdateUserNameEmail(authedUser); err != nil {
			//Failed to update user info
			logging.WriteLog(logging.LogLevelError, "authrouters/AuthCallback", authedUser.GetCompositeID(), logging.ResultFailure, []string{"Account update failed with database error", err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Internal error occured updating account info.", "authError")
			return
		}
		logging.WriteLog(logging.LogLevelDebug, "authrouters/AuthCallback", authedUser.GetCompositeID(), logging.ResultInfo, []string{"Name/email updated!"})
		databaseUser, err = database.DBInterface.GetUser(authedUser)
		if err != nil {
			//Failed to update user info
			logging.WriteLog(logging.LogLevelError, "authrouters/AuthCallback", authedUser.GetCompositeID(), logging.ResultFailure, []string{"Account retrieval after update failed with database error", err.Error()})
			redirectWithFlash(responseWriter, request, "/", "Internal error occured updating account info.", "authError")
			return
		}
	}

	//Set logon data to session
	session.Values["oidcuserinfo"] = databaseUser

	err = session.Save(request, responseWriter)
	if err != nil {
		logging.WriteLog(logging.LogLevelError, "authrouters/AuthCallback", authedUser.GetCompositeID(), logging.ResultFailure, []string{"Failed to save session", err.Error()})
	} else {
		logging.WriteLog(logging.LogLevelInfo, "authrouters/AuthCallback", authedUser.GetCompositeID(), logging.ResultSuccess, []string{"Logged in", "email", claims.Email, "subject", idToken.Subject, "name", claims.Username})
	}
	//Redirect user to main page
	http.Redirect(responseWriter, request, "/", http.StatusFound)
}
