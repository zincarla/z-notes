package routers

import (
	"html"
	"html/template"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"z-notes/config"
	"z-notes/database"
	"z-notes/interfaces"
	"z-notes/logging"

	"github.com/gorilla/mux"
)

//UploadFilePostRouter serves requests to /uploadpage
func UploadFilePostRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]

	if !TemplateInput.IsLoggedOn() {
		//Error with no logon
		redirectWithFlash(responseWriter, request, "/", "You must be logged in to perform that action", "uploadError")
		return
	}

	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "uploadpage/UploadFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Form filled incorrectly", "uploadError")
		return
	}
	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	//Check user permissions
	access, err = database.DBInterface.GetEffectivePermission(access)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "uploadpage/UploadFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "uploadError")
		return
	}
	if !access.Access.HasAccess(interfaces.Write) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "uploadError")
		return
	}

	//Parse Upload
	err = request.ParseMultipartForm(config.Configuration.MaxUploadBytes)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "uploadpage/UploadFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error parsing upload files", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Error uploading files", "uploadError")
		return
	}
	fileHeaders := request.MultipartForm.File["Files"]
	if len(fileHeaders) <= 0 {
		//Error with no upload
		redirectWithFlash(responseWriter, request, "/", "Empty upload form submitted", "uploadError")
		return
	}

	returnMessage := ""
	for _, fileHeader := range fileHeaders {
		fileStream, err := fileHeader.Open()
		if err != nil {
			logging.WriteLog(logging.LogLevelWarning, "uploadpage/UploadFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"User failed to upload file", fileHeader.Filename, err.Error()})
			returnMessage += "Failed to upload file " + html.EscapeString(fileHeader.Filename) + "<br>"
		} else {
			filePath, err := handleFileUpload(PageID, &fileStream, fileHeader)
			if err != nil {
				logging.WriteLog(logging.LogLevelWarning, "uploadpage/UploadFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error saving file", fileHeader.Filename, err.Error()})
				returnMessage += "Failed to upload file " + html.EscapeString(fileHeader.Filename) + "<br>"
			} else if request.FormValue("AutoAddFile") == "checked" {
				logging.WriteLog(logging.LogLevelDebug, "uploadpage/UploadFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Not implemented", filePath})
				//Grab page content
				pageData, err := database.DBInterface.GetPage(PageID)
				if err != nil {
					logging.WriteLog(logging.LogLevelWarning, "uploadpage/UploadFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting page data", pageID, err.Error()})
					returnMessage += "Failed to add " + fileHeader.Filename + " to page content<br>"
				} else {
					//Add
					//TODO: Change add method depending on file type. For now, just links

					mimeType := fileHeader.Header.Get("Content-Type")
					logging.WriteLog(logging.LogLevelDebug, "uploadpage/UploadFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultInfo, []string{"Mime is", mimeType})

					mimeType = strings.Split(mimeType, "/")[0]

					switch mimeType {
					case "image":
						pageData.Content = pageData.Content + "\r\n\r\n![Uploaded Image: " + fileHeader.Filename + "](./resources/" + fileHeader.Filename + ")\r\n"
						err = database.DBInterface.UpdatePage(pageData)
						if err != nil {
							logging.WriteLog(logging.LogLevelWarning, "uploadpage/UploadFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured updating page data", pageID, err.Error()})
							returnMessage += "Failed to add " + fileHeader.Filename + " to page content<br>"
						}
					default:
						pageData.Content = pageData.Content + "\r\n\r\n[Uploaded File: " + fileHeader.Filename + "](./resources/" + fileHeader.Filename + ")\r\n"
						err = database.DBInterface.UpdatePage(pageData)
						if err != nil {
							logging.WriteLog(logging.LogLevelWarning, "uploadpage/UploadFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured updating page data", pageID, err.Error()})
							returnMessage += "Failed to add " + fileHeader.Filename + " to page content<br>"
						}
					}
				}
			}
		}
		//Close the stream before next iteration of loop.
		fileStream.Close()
	}
	if returnMessage != "" {
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/file", "Files uploaded, with issues.<br>"+returnMessage, "uploadFinish")
	} else {
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/file", "Files uploaded successfully", "uploadFinish")
	}
	return
}

//UploadFileGetRouter serves requests to /uploadpage/{pageID}
func UploadFileGetRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]

	if !TemplateInput.IsLoggedOn() {
		//Error with no logon
		redirectWithFlash(responseWriter, request, "/", "You must be logged in to perform that action", "uploadError")
		return
	}

	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "uploadpage/UploadFileGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Form filled incorrectly", "uploadError")
		return
	}
	TemplateInput.PageData.ID = PageID
	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	//Check user permissions
	access, err = database.DBInterface.GetEffectivePermission(access)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "uploadpage/UploadFileGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "uploadError")
		return
	}
	if !access.Access.HasAccess(interfaces.Write | interfaces.Read) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "uploadError")
		return
	}

	//Grab child pages so that the menu may be constructed in template
	children, err := database.DBInterface.GetPageChildren(PageID)
	if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "editpage/EditPageGetRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get child pages", err.Error()})
		TemplateInput.HTMLMessage = template.HTML("Failed to get child pages, internal error occured.")
	} else {
		TemplateInput.ChildPages = children
	}

	resources, err := getPageResources(PageID)
	if err != nil {
		TemplateInput.HTMLMessage = template.HTML("Failed to get page resources")
	}
	TemplateInput.PageResources = resources

	//Send in template
	replyWithTemplate("uploadpage.html", TemplateInput, responseWriter, request)
}

//DeleteFilePostRouter serves requests to /deletefile
func DeleteFilePostRouter(responseWriter http.ResponseWriter, request *http.Request) {
	TemplateInput := getNewTemplateInput(responseWriter, request)
	urlVariables := mux.Vars(request)
	pageID := urlVariables["pageID"]

	if !TemplateInput.IsLoggedOn() {
		//Error with no logon
		redirectWithFlash(responseWriter, request, "/", "You must be logged in to perform that action", "deleteError")
		return
	}

	//Convert PageID
	PageID, err := strconv.ParseUint(pageID, 10, 64)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "uploadpage/DeleteFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured parsing pageID", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Form filled incorrectly", "deleteError")
		return
	}
	//Check permissions
	access := interfaces.UserPageAccess{PageID: PageID, User: TemplateInput.UserInformation}
	//Check user permissions
	access, err = database.DBInterface.GetEffectivePermission(access)
	if err != nil {
		//If any error occurs, log it and respond with redirect
		logging.WriteLog(logging.LogLevelWarning, "uploadpage/DeleteFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Error occured getting user permissions", request.FormValue("PageID"), err.Error()})
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "deleteError")
		return
	}
	if !access.Access.HasAccess(interfaces.Delete) {
		redirectWithFlash(responseWriter, request, "/", "Access Denied", "deleteError")
		return
	}

	//Parse delete request
	if request.FormValue("File") == "" {
		logging.WriteLog(logging.LogLevelWarning, "uploadpage/DeleteFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Form missing file to delete"})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/file", "No file to delete selected", "deleteError")
		return
	}

	//Verify file
	filePath := getPageResourcePath(PageID, request.FormValue("File"))
	_, err = os.Stat(filePath)
	if err != nil && os.IsNotExist(err) {
		logging.WriteLog(logging.LogLevelWarning, "uploadpage/DeleteFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to delete file, it does not exist", pageID, request.FormValue("File"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/file", "File does not exist", "deleteError")
		return
	} else if err != nil {
		logging.WriteLog(logging.LogLevelWarning, "uploadpage/DeleteFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to delete file", pageID, request.FormValue("File"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/file", "Internal error deleting file", "deleteError")
		return
	}

	//Now delete
	if err := os.Remove(filePath); err != nil {
		logging.WriteLog(logging.LogLevelWarning, "uploadpage/DeleteFilePostRouter", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to delete file", pageID, request.FormValue("File"), err.Error()})
		redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/file", "Internal error deleting file", "deleteError")
		return
	}

	//Return success by redirect
	redirectWithFlash(responseWriter, request, "/page/"+strconv.FormatUint(PageID, 10)+"/file", "File deleted successfully", "deleteSuccess")
	return
}

//handleFileUpload handles a requested file upload, returns the local url of the file, and an error if failed
func handleFileUpload(PageID uint64, uploadedFile *multipart.File, uploadedFileHeader *multipart.FileHeader) (string, error) {
	//Get page data
	_, err := database.DBInterface.GetPage(PageID)
	if err != nil {
		return "", err
	}
	//Check if folder path exists
	_, err = os.Stat(getPageResourceRootPath(PageID))
	if err != nil && os.IsNotExist(err) {
		//Create folder if necessary
		err = os.Mkdir(getPageResourceRootPath(PageID), 0750)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		//Other errors usually represent errors with the path/device/permissions. Doesn't matter, we will error out here
		return "", err
	}

	//Get what the system file path should be
	filePath := getPageResourcePath(PageID, uploadedFileHeader.Filename)

	//Now we copy the file
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0660)
	defer f.Close()
	defer (*uploadedFile).Close()
	if err != nil {
		return "", err
	}
	_, err = io.Copy(f, *uploadedFile)
	return ("./resources/" + uploadedFileHeader.Filename), err
}

//getPageResources returns a slice of resources for a page
func getPageResources(PageID uint64) ([]string, error) {
	//Check if folder path exists
	resourcePath := getPageResourceRootPath(PageID)
	_, err := os.Stat(resourcePath)
	if err != nil && os.IsNotExist(err) {
		return nil, nil //In this case, gobble as it is expected some pages will not have resources and resource directories
	} else if err != nil {
		//Other errors usually represent errors with the path/device/permissions. Doesn't matter, we will error out here
		return nil, err
	}

	//Create Path
	var resources []string
	files, err := ioutil.ReadDir(resourcePath)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if f.IsDir() == false {
			resources = append(resources, f.Name())
		}
	}
	return resources, nil
}
