package routers

import (
	"errors"
	"html/template"
	"math"
	"net"
	"net/http"
	"strconv"
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
	MovingParentPageData  interfaces.Page
	SearchResults         []interfaces.Page
	PageMenu              template.HTML
	BreadCrumbRoot        interfaces.Page
	AllowAccountCreation  bool
	AccountRequiredToView bool
	RedirectLink          string
	CSRF                  template.HTML
	UserInformation       interfaces.UserInformation
	PageResources         []string

	PagePermissions []interfaces.UserPageAccess
	UserTokens      []interfaces.APITokenInformation

	//RequestStart is start time for a user request
	RequestStart time.Time
	//RequestTime is time it took to process a user request in MS
	RequestTime int64
}

func (ti templateInput) IsLoggedOn() bool {
	return ti.UserInformation.OIDCSubject != "" && ti.UserInformation.OIDCIssuer != ""
}

func (ti templateInput) ParseMarkdown(markdownContent string) template.HTML {
	parsedContent, err := GetParsedPage(markdownContent)
	if err != nil {
		parsedContent = template.HTML("")
		logging.LogInterface.WriteLog(logging.LogLevelError, "routertemplate/ParseMarkdown", "", logging.ResultFailure, []string{"failed to parse provided content from template", markdownContent, err.Error()})
	}
	return parsedContent
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
	TemplateInput := templateInput{Title: "Z-Notes",
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

//GeneratePageMenu generates a template.HTML menu given a few numbers. Returns a menu like "<< 1, 2, 3, [4], 5, 6, 7 >>"
func GeneratePageMenu(Offset int64, Stride int64, Max int64, PageURL string) (template.HTML, error) {
	//Validate parameters
	if Offset < 0 || Stride <= 0 || Max < 0 || Offset > Max {
		return template.HTML(""), errors.New("Parameters must be positive numbers")
	}
	if Max == 0 {
		return template.HTML("1"), nil
	}

	//Jump to top of results
	ToReturn := "<a href=\"" + PageURL + "\">&#x3C;&#x3C;</a>"
	//Max possible page number
	maxPage := int64(math.Ceil(float64(Max) / float64(Stride)))
	lastPage := maxPage
	//Current page number
	currentPage := int64(math.Floor(float64(Offset)/float64(Stride)) + 1)
	//Minimum page number we will show
	minPage := currentPage - 3
	if minPage < 1 {
		minPage = 1
	}
	if maxPage > currentPage+3 {
		maxPage = currentPage + 3
	}

	for processPage := minPage; processPage <= maxPage; processPage++ {
		if processPage != currentPage {
			ToReturn = ToReturn + ", <a href=\"" + PageURL + "?searchPage=" + strconv.FormatInt(processPage, 10) + "\">" + strconv.FormatInt(processPage, 10) + "</a>"
		} else {
			ToReturn = ToReturn + ", " + strconv.FormatInt(currentPage, 10)
		}
	}

	//Add end
	ToReturn = ToReturn + ", <a href=\"" + PageURL + "?searchPage=" + strconv.FormatInt(lastPage, 10) + "\">&#x3E;&#x3E;</a>"
	return template.HTML(ToReturn), nil
}
