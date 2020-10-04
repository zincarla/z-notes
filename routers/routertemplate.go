package routers

import (
	"html/template"
	"net"
	"net/http"
	"time"
	"z-notes/config"
	"z-notes/interfaces"
	"z-notes/logging"
	"z-notes/routers/templatecache"

	"github.com/gorilla/csrf"
)

type templateInput struct {
	Title                 string
	Version               string
	HTMLMessage           template.HTML
	PageContent           template.HTML
	PageData              interfaces.Page
	ParentPageData        interfaces.Page
	MovingParentPageData  interfaces.Page
	ChildPages            []interfaces.Page
	SearchResults         []interfaces.Page
	AllowAccountCreation  bool
	AccountRequiredToView bool
	RedirectLink          string
	CSRF                  template.HTML
	UserInformation       interfaces.UserInformation
	PageResources         []string

	PagePermissions []interfaces.UserPageAccess

	//RequestStart is start time for a user request
	RequestStart time.Time
	//RequestTime is time it took to process a user request in MS
	RequestTime int64
}

func (ti templateInput) IsLoggedOn() bool {
	return ti.UserInformation.OIDCSubject != "" && ti.UserInformation.OIDCIssuer != ""
}

func replyWithTemplate(templateName string, templateInputInterface interface{}, responseWriter http.ResponseWriter, request *http.Request) {
	//Call Template
	templateToUse := templatecache.TemplateCache
	if ti, ok := templateInputInterface.(templateInput); ok {
		ti.RequestTime = time.Now().Sub(ti.RequestStart).Nanoseconds() / 1000000 //Nanosecond to Millisecond
		applyFlash(responseWriter, request, &ti)
		templateInputInterface = ti
	}
	err := templateToUse.ExecuteTemplate(responseWriter, templateName, templateInputInterface)
	if err != nil {
		logging.WriteLog(logging.LogLevelCritical, "routertemplate/replyWithTemplate", "*", logging.ResultFailure, []string{"Parse Error", err.Error()})
		http.Error(responseWriter, "", http.StatusInternalServerError)
		return
	}
}

//getNewTemplateInput helper function initiliazes a new templateInput with common information
func getNewTemplateInput(responseWriter http.ResponseWriter, request *http.Request) templateInput {
	TemplateInput := templateInput{Title: "ZNotes",
		Version:              config.ApplicationVersion,
		AllowAccountCreation: config.Configuration.AllowAccountCreation,
		RequestStart:         time.Now(),
		CSRF:                 csrf.TemplateField(request)}

	session, err := config.SessionStore.Get(request, config.SessionVariableName)
	if err == nil {
		//Load user session
		if oidcUserInfo, ok := session.Values["oidcuserinfo"].(interfaces.UserInformation); ok {
			if config.Configuration.OpenIDLogonExpireTime > 0 {
				if oidcUserInfo.OIDCIssueTime.Add(time.Second * time.Duration(config.Configuration.OpenIDLogonExpireTime)).After(time.Now()) {
					TemplateInput.UserInformation = oidcUserInfo
				} //Else expired
			} else {
				TemplateInput.UserInformation = oidcUserInfo
			}
		}
	}

	//Add IP to user info
	TemplateInput.UserInformation.IP, _, err = net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		TemplateInput.UserInformation.IP = request.RemoteAddr
	}

	return TemplateInput
}

//applyFlash checks for flash cookies, and applies them to template
func applyFlash(responseWriter http.ResponseWriter, request *http.Request, TemplateInput *templateInput) {
	if request.FormValue("flash") == "" {
		return
	}
	session, err := config.SessionStore.Get(request, config.SessionVariableName)
	if err == nil {
		//Load flash if necessary
		if request.FormValue("flash") != "" {
			pendingFlashes := session.Flashes(request.FormValue("flash"))
			if len(pendingFlashes) > 0 {
				fullMessage := ""
				for _, pendingFlash := range pendingFlashes {
					if pf, ok := pendingFlash.(string); ok {
						fullMessage += pf + "<br>"
					}
				}
				TemplateInput.HTMLMessage += template.HTML(fullMessage)
				session.Save(request, responseWriter)
			}
		}
	}
}

//createFlash creates a flash cookie and saves the session
func createFlash(responseWriter http.ResponseWriter, request *http.Request, flashMessage string, flashName string) error {
	session, _ := config.SessionStore.Get(request, config.SessionVariableName)
	session.AddFlash(flashMessage, flashName)
	return session.Save(request, responseWriter)
}

//creates a flash cookie, and sends a redirect to the client to the root with the flash cookie message
func redirectWithFlash(responseWriter http.ResponseWriter, request *http.Request, redirectURL string, flashMessage string, flashName string) error {
	err := createFlash(responseWriter, request, flashMessage, flashName)
	http.Redirect(responseWriter, request, redirectURL+"?flash="+flashName, http.StatusFound)
	return err
}
