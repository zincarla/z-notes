package routers

import (
	"html/template"
	"z-notes/database"
	"z-notes/interfaces"
	"z-notes/logging"
)

//GetCrumbs generates a page that chains to the pageID and includes each page's children. Used for generating menus in html templates
func GetCrumbs(TemplateInput *templateInput, addPageChildren bool) error {

	//Grab RootPage, this will be start of crumbs
	RootPage := interfaces.Page{Name: "Library Root", OwnerID: TemplateInput.UserInformation.DBID}

	//Grab path for breadcrumbs
	pagePath, err := database.DBInterface.GetPagePath(TemplateInput.PageData.ID, true)
	if err != nil {
		return err
	}

	//Now we have path, we must fill in children
	//First fill in root children
	RootPage.Children, err = database.DBInterface.GetRootPages(TemplateInput.UserInformation.DBID)
	if err != nil {
		return err
	}
	//Now fill in children from path
	for i := 0; i < len(pagePath); i++ {
		pagePath[i].Children, err = database.DBInterface.GetPageChildren(pagePath[i].ID)
		if err != nil {
			return err
		}
	}
	//Finally attach it all together
	//First, we go through pagePath in reverse order
	for i := len(pagePath) - 1; i > 0; i-- {
		//We grab the current pages parent (One back in the slice)
		//And iterate through it's children to find the current page's ID
		for ci := 0; ci < len(pagePath[i-1].Children); ci++ {
			if pagePath[i-1].Children[ci].ID == pagePath[i].ID {
				//Once we find it, we replace the child with the more completed current page
				//Effectivly we a popping the end off the slice, and appending it to the previous item in the slice
				//This will give us a tree of Pages where each page is nested in the previous page
				pagePath[i-1].Children[ci] = pagePath[i]
				break //Break from the outer loop
			}
		}
	}
	//Finally, we attach this tree to the root page
	for ci := 0; ci < len(RootPage.Children); ci++ {
		if RootPage.Children[ci].ID == pagePath[0].ID {
			RootPage.Children[ci] = pagePath[0]
			break
		}
	}
	//At this point, RootPage should look like this
	/*
		RootPage.Children
		-SomePage
		-SomePage
		-SomePageInCrumbPath.Children
		--SomePage
		--SomePageInCrumbPath.Children
		---SomePage....
		-SomePage
	*/

	if addPageChildren {
		TemplateInput.PageData.Children = pagePath[len(pagePath)-1].Children
	}

	TemplateInput.BreadCrumbRoot = RootPage
	return nil
}

//FillTemplatePageData fills the templates page data, crumbs, access is not verified!
func FillTemplatePageData(PageID uint64, TemplateInput *templateInput) error {
	//Grab page content
	pageData, err := database.DBInterface.GetPage(PageID)
	if err != nil {
		return err
	}

	//Add content to template
	TemplateInput.Title = pageData.Name
	TemplateInput.PageData = pageData

	//Grab path for breadcrumbs
	err = GetCrumbs(TemplateInput, true)
	if err != nil {
		logging.WriteLog(logging.LogLevelError, "templateinputhelpers/FillTemplatePageData", TemplateInput.UserInformation.GetCompositeID(), logging.ResultFailure, []string{"Failed to get page crumbs from database", err.Error()})
		TemplateInput.HTMLMessage += template.HTML("Failed to get page crumbs, internal error occured.")
	}

	return nil
}
