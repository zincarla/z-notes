package mariadbplugin

import (
	"database/sql"
	"errors"
	"fmt"
	"z-notes/interfaces"
)

//UpdateTokenPermission creates or updates a pagepermission for tokens
func (DBConnection *MariaDBPlugin) UpdateTokenPermission(permission interfaces.TokenPageAccess) error {
	if permission.PageID == 0 {
		return errors.New("Page ID not provided")
	}
	if permission.Token.ID == 0 {
		return errors.New("Token ID not provided")
	}
	if permission.Access == 0 {
		return errors.New("Access not provided")
	}

	query := `INSERT INTO PageTokenPermissions (PageID, TokenID, Permissions) VALUES (?, ?, ?) 
				ON DUPLICATE KEY UPDATE
				Permissions=VALUES(Permissions);`

	//And apply
	_, err := DBConnection.DBHandle.Exec(query, permission.PageID, permission.Token.ID, permission.Access)
	return err
}

//RemoveTokenPermission removes a PagePermission for a token (error nil on success)
func (DBConnection *MariaDBPlugin) RemoveTokenPermission(permissionID uint64) error {
	_, err := DBConnection.DBHandle.Exec("DELETE FROM PageTokenPermissions WHERE ID = ?", permissionID)
	return err
}

//GetTokenPermissions returns the token permissions assigned directly to a page with the given id
func (DBConnection *MariaDBPlugin) GetTokenPermissions(pageID uint64) ([]interfaces.TokenPageAccess, error) {
	var toReturn []interfaces.TokenPageAccess
	if pageID == 0 {
		return toReturn, errors.New("Page ID not provided")
	}

	//run the query
	rows, err := DBConnection.DBHandle.Query("SELECT ID, TokenID, Permissions FROM PageTokenPermissions WHERE PageID=?", pageID)
	if err != nil {
		return toReturn, err
	}
	defer rows.Close()

	//For each row
	for rows.Next() {
		toAdd := interfaces.TokenPageAccess{PageID: pageID}
		//Parse out the data
		err := rows.Scan(&toAdd.ID, &toAdd.Token.ID, &toAdd.Access)
		if err != nil {
			return toReturn, err
		}
		//Add this result to ToReturn
		toReturn = append(toReturn, toAdd)
	}

	return toReturn, nil
}

//GetTokenPermission returns the token permission assigned directly to a page
func (DBConnection *MariaDBPlugin) GetTokenPermission(pageAccess interfaces.TokenPageAccess) (interfaces.TokenPageAccess, error) {
	toReturn := pageAccess

	//Prefer main ID if provided
	if pageAccess.ID != 0 {
		//run the query
		err := DBConnection.DBHandle.QueryRow("SELECT ID, TokenID, PageID, Permissions FROM PageTokenPermissions WHERE ID=?", pageAccess.ID).Scan(&toReturn.ID, &toReturn.Token.ID, &toReturn.PageID, &toReturn.Access)
		if err != nil {
			return toReturn, err
		}
	} else {
		//Fallback to PageID+DBID combo
		if pageAccess.PageID == 0 {
			return toReturn, errors.New("Page ID not provided, nor was ID")
		}
		if pageAccess.Token.ID == 0 {
			return toReturn, errors.New("Token ID not provided, nor was ID")
		}

		//run the query
		err := DBConnection.DBHandle.QueryRow("SELECT ID, TokenID, PageID, Permissions FROM PageTokenPermissions WHERE PageID=? AND TokenID=?", pageAccess.PageID, pageAccess.Token.ID).Scan(&toReturn.ID, &toReturn.Token.ID, &toReturn.PageID, &toReturn.Access)
		if err != nil {
			return toReturn, err
		}
	}

	return toReturn, nil
}

//GetEffectiveTokenPermission returns the effective permissions for a token on a page, this takes into account inherited permissions
func (DBConnection *MariaDBPlugin) GetEffectiveTokenPermission(pageAccess interfaces.TokenPageAccess) (interfaces.TokenPageAccess, error) {
	toReturn := interfaces.TokenPageAccess{PageID: pageAccess.PageID, Token: pageAccess.Token}
	if pageAccess.PageID == 0 {
		return toReturn, errors.New("Page ID not provided")
	}
	if pageAccess.Token.ID == 0 {
		return toReturn, errors.New("Token ID not provided")
	}

	//Starting from the root, work down to the current page
	//Inherited permissions build on each-other
	//Denials can be used to one-off prevent certain inherited permissions, or can cancel out an inherited permission from then down if inherited

	//First we need the page path
	pagePath, err := DBConnection.GetPagePath(pageAccess.PageID, false)
	if err != nil {
		return toReturn, fmt.Errorf("Failed to generate pagepath for permissions for page %v", pageAccess.PageID)
	}

	var subAccess interfaces.TokenPageAccess

	//Pagepath is in reverse order
	for index := len(pagePath) - 1; index >= 0; index-- {
		//Get the user's permissions at this page
		subAccess, err = DBConnection.GetTokenPermission(interfaces.TokenPageAccess{PageID: pagePath[index].ID, Token: interfaces.APITokenInformation{ID: pageAccess.Token.ID}})
		if err != nil {
			if err != sql.ErrNoRows { //Return an error, but only if the error is not that we found no permissions
				return toReturn, fmt.Errorf("Failed to get sub permissions for page %v and token %v", pagePath[index].ID, pageAccess.Token.ID)
			}
			subAccess = interfaces.TokenPageAccess{}
		}

		//Only process this permission if inherited
		if (subAccess.Access.HasAccess(interfaces.Inherits) || subAccess.PageID == pageAccess.PageID) && subAccess.ID != 0 {
			//If inherits, add to current results, if denial, remove from current results
			if subAccess.Access.HasAccess(interfaces.Deny) {
				//Remove permissions
				toReturn.Access = toReturn.Access & (^subAccess.Access)
			} else {
				//Add permissions
				toReturn.Access = toReturn.Access | subAccess.Access
			}
		}
	}

	//Remove the deny flag if applied
	toReturn.Access = toReturn.Access & (^interfaces.Inherits)
	toReturn.Access = toReturn.Access & (^interfaces.Deny)

	return toReturn, nil
}
