package mariadbplugin

import (
	"database/sql"
	"errors"
	"fmt"
	"z-notes/interfaces"
)

//UpdatePermission creates or updates a pagepermission
func (DBConnection *MariaDBPlugin) UpdatePermission(permission interfaces.UserPageAccess) error {
	if permission.PageID == 0 {
		return errors.New("Page ID not provided")
	}
	if permission.User.DBID == 0 {
		return errors.New("User ID not provided")
	}
	if permission.Access == 0 {
		return errors.New("Access not provided")
	}

	query := `INSERT INTO PagePermissions (PageID, UserID, Permissions) VALUES (?, ?, ?) 
				ON DUPLICATE KEY UPDATE
				Permissions=VALUES(Permissions);`

	//And apply
	_, err := DBConnection.DBHandle.Exec(query, permission.PageID, permission.User.DBID, permission.Access)
	return err
}

//RemovePermission removes a PagePermission (error nil on success)
func (DBConnection *MariaDBPlugin) RemovePermission(permissionID uint64) error {
	_, err := DBConnection.DBHandle.Exec("DELETE FROM PagePermissions WHERE ID = ?", permissionID)
	return err
}

//GetPermissions returns the permissions assigned directly to a page with the given id
func (DBConnection *MariaDBPlugin) GetPermissions(pageID uint64) ([]interfaces.UserPageAccess, error) {
	var toReturn []interfaces.UserPageAccess
	if pageID == 0 {
		return toReturn, errors.New("Page ID not provided")
	}

	//run the query
	rows, err := DBConnection.DBHandle.Query("SELECT ID, UserID, Permissions FROM PagePermissions WHERE PageID=?", pageID)
	if err != nil {
		return toReturn, err
	}
	defer rows.Close()

	//For each row
	for rows.Next() {
		toAdd := interfaces.UserPageAccess{PageID: pageID}
		//Parse out the data
		err := rows.Scan(&toAdd.ID, &toAdd.User.DBID, &toAdd.Access)
		if err != nil {
			return toReturn, err
		}
		//Add this result to ToReturn
		toReturn = append(toReturn, toAdd)
	}

	return toReturn, nil
}

//GetPermission returns the permission assigned directly to a page for a user
func (DBConnection *MariaDBPlugin) GetPermission(pageAccess interfaces.UserPageAccess) (interfaces.UserPageAccess, error) {
	toReturn := pageAccess

	//Prefer main ID if provided
	if pageAccess.ID != 0 {
		//run the query
		err := DBConnection.DBHandle.QueryRow("SELECT ID, UserID, PageID, Permissions FROM PagePermissions WHERE ID=?", pageAccess.ID).Scan(&toReturn.ID, &toReturn.User.DBID, &toReturn.PageID, &toReturn.Access)
		if err != nil {
			return toReturn, err
		}
	} else {
		//Fallback to PageID+DBID combo
		if pageAccess.PageID == 0 {
			return toReturn, errors.New("Page ID not provided, nor was ID")
		}
		if pageAccess.User.DBID == 0 {
			return toReturn, errors.New("User ID not provided, nor was ID")
		}

		//run the query
		err := DBConnection.DBHandle.QueryRow("SELECT ID, UserID, PageID, Permissions FROM PagePermissions WHERE PageID=? AND UserID=?", pageAccess.PageID, pageAccess.User.DBID).Scan(&toReturn.ID, &toReturn.User.DBID, &toReturn.PageID, &toReturn.Access)
		if err != nil {
			return toReturn, err
		}
	}

	return toReturn, nil
}

//GetEffectivePermission returns the effective permissions for a user on a page, this takes into account inherited permissions
func (DBConnection *MariaDBPlugin) GetEffectivePermission(pageAccess interfaces.UserPageAccess) (interfaces.UserPageAccess, error) {
	toReturn := interfaces.UserPageAccess{PageID: pageAccess.PageID, User: pageAccess.User}
	if pageAccess.PageID == 0 {
		return toReturn, errors.New("Page ID not provided")
	}
	if pageAccess.User.DBID == 0 {
		return toReturn, errors.New("User ID not provided")
	}

	//Starting from the root, work down to the current page
	//Inherited permissions build on each-other
	//Denials can be used to one-off prevent certain inherited permissions, or can cancel out an inherited permission from then down if inherited

	//First we shortcut if the page is owned by the user
	page, err := DBConnection.GetPage(pageAccess.PageID)
	if err != nil {
		return toReturn, fmt.Errorf("Failed to get page for permissions for page %v", pageAccess.PageID)
	}
	//Shortcut, if user is the page owner, they always have full control
	if page.OwnerID == pageAccess.User.DBID {
		pageAccess.Access = interfaces.Full
		return pageAccess, nil
	}

	//First we need the page path
	pagePath, err := DBConnection.GetPagePath(pageAccess.PageID)
	if err != nil {
		return toReturn, fmt.Errorf("Failed to generate pagepath for permissions for page %v", pageAccess.PageID)
	}

	//Pagepath is in reverse order
	var subAccessSlice []interfaces.UserPageAccess
	if pageAccess.User.DBID != interfaces.AnonymousUserID && pageAccess.User.DBID != interfaces.AuthenticatedUserID {
		subAccessSlice = make([]interfaces.UserPageAccess, 2)
	} else {
		subAccessSlice = make([]interfaces.UserPageAccess, 1)
	}
	for index := len(pagePath) - 1; index >= 0; index-- {
		//Get the user's permissions at this page
		subAccessSlice[0], err = DBConnection.GetPermission(interfaces.UserPageAccess{PageID: pagePath[index].ID, User: interfaces.UserInformation{DBID: pageAccess.User.DBID}})
		if err != nil {
			if err != sql.ErrNoRows { //Return an error, but only if the error is not that we found no permissions
				return toReturn, fmt.Errorf("Failed to get sub permissions for page %v and user %v", pagePath[index].ID, pageAccess.User.DBID)
			}
			subAccessSlice[0] = interfaces.UserPageAccess{}
		}
		if pageAccess.User.DBID != interfaces.AnonymousUserID && pageAccess.User.DBID != interfaces.AuthenticatedUserID {
			subAccessSlice[1], err = DBConnection.GetPermission(interfaces.UserPageAccess{PageID: pagePath[index].ID, User: interfaces.UserInformation{DBID: interfaces.AuthenticatedUserID}})
			if err != nil {
				if err != sql.ErrNoRows { //Return an error, but only if the error is not that we found no permissions
					return toReturn, fmt.Errorf("Failed to get sub permissions for authenticated user and page %v", pagePath[index].ID)
				}
				subAccessSlice[1] = interfaces.UserPageAccess{}
			}
		}

		//Sort for denial last
		if len(subAccessSlice) > 1 && subAccessSlice[0].Access.HasAccess(interfaces.Deny) {
			subAccessSlice[0], subAccessSlice[1] = subAccessSlice[1], subAccessSlice[0]
		}

		for index := 0; index < len(subAccessSlice); index++ {
			//Only process this permission if inherited
			if (subAccessSlice[index].Access.HasAccess(interfaces.Inherits) || subAccessSlice[index].PageID == pageAccess.PageID) && subAccessSlice[index].ID != 0 {
				//If inherits, add to current results, if denial, remove from current results
				if subAccessSlice[index].Access.HasAccess(interfaces.Deny) {
					//Remove permissions
					toReturn.Access = toReturn.Access & (^subAccessSlice[index].Access)
				} else {
					//Add permissions
					toReturn.Access = toReturn.Access | subAccessSlice[index].Access
				}
			}
		}
	}

	//Remove the deny flag if applied
	toReturn.Access = toReturn.Access & (^interfaces.Inherits)
	toReturn.Access = toReturn.Access & (^interfaces.Deny)

	return toReturn, nil
}
