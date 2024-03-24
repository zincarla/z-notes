package mariadbplugin

import (
	"errors"
	"z-notes/interfaces"

	"github.com/go-sql-driver/mysql"
)

//CreatePage is used to create a new page (return nil on success)
func (DBConnection *MariaDBPlugin) CreatePage(pageData interfaces.Page) (uint64, error) {
	//Must have ownerid
	if pageData.OwnerID == 0 {
		return 0, errors.New("OwnerID information not provided")
	}
	query := "INSERT INTO Pages (Name, OwnerID, PrevID, Content) VALUES (?, ?, ?, ?);"
	queryArray := []interface{}{pageData.Name, pageData.OwnerID, pageData.PrevID, pageData.Content}
	if pageData.PrevID == 0 {
		query = "INSERT INTO Pages (Name, OwnerID, Content) VALUES (?, ?, ?);"
		queryArray = []interface{}{pageData.Name, pageData.OwnerID, pageData.Content}
	}

	resultInfo, err := DBConnection.DBHandle.Exec(query, queryArray...)
	if err != nil {
		return 0, err
	}
	id, err := resultInfo.LastInsertId()
	return uint64(id), err
}

//UpdatePage updates a page
func (DBConnection *MariaDBPlugin) UpdatePage(pageData interfaces.Page) error {
	if pageData.ID == 0 {
		return errors.New("Page ID not provided")
	}
	query := "UPDATE Pages SET"
	queryArray := []interface{}{}
	if pageData.Name != "" {
		query = query + " Name=?,"
		queryArray = append(queryArray, pageData.Name)
	}

	//PrevID
	if pageData.PrevID != 0 {
		query = query + " PrevID=?,"
		queryArray = append(queryArray, pageData.PrevID)
	} else {
		query = query + " PrevID=NULL,"
	}

	//Content
	query = query + " Content=?,"
	queryArray = append(queryArray, pageData.Content)

	query = query[:len(query)-1] //Trim last comma

	//Finish query
	query = query + " WHERE ID=?"
	queryArray = append(queryArray, pageData.ID)

	//And apply
	_, err := DBConnection.DBHandle.Exec(query, queryArray...)
	return err
}

//RemovePage removes a page (error nil on success)
func (DBConnection *MariaDBPlugin) RemovePage(pageID uint64) error {
	_, err := DBConnection.DBHandle.Exec("DELETE FROM Pages WHERE ID = ?", pageID)
	return err
}

//GetPage returns a page's data
func (DBConnection *MariaDBPlugin) GetPage(pageID uint64) (interfaces.Page, error) {
	toReturn := interfaces.Page{ID: pageID}

	query := "SELECT ID, Name, OwnerID, PrevID, Content FROM Pages WHERE ID=?"
	queryArray := []interface{}{}
	queryArray = append(queryArray, pageID)

	var NPrevID NullUint64
	err := DBConnection.DBHandle.QueryRow(query, queryArray...).Scan(&toReturn.ID, &toReturn.Name, &toReturn.OwnerID, &NPrevID, &toReturn.Content)
	if err != nil {
		return toReturn, err
	}
	if NPrevID.Valid {
		toReturn.PrevID = NPrevID.Uint64
	}

	return toReturn, nil
}

//GetPageChildren returns incomplete page data for children of the specified page (Content not included)
func (DBConnection *MariaDBPlugin) GetPageChildren(pageID uint64) ([]interfaces.Page, error) {
	var toReturn []interfaces.Page
	if pageID == 0 {
		return toReturn, errors.New("Page ID not provided")
	}
	query := "SELECT ID, Name, OwnerID FROM Pages WHERE PrevID=?"
	queryArray := []interface{}{}
	queryArray = append(queryArray, pageID)

	//Now we have query and args, run the query
	rows, err := DBConnection.DBHandle.Query(query, queryArray...)
	if err != nil {
		return toReturn, err
	}
	defer rows.Close()

	//For each row
	for rows.Next() {
		toAdd := interfaces.Page{PrevID: pageID}
		//Parse out the data
		err := rows.Scan(&toAdd.ID, &toAdd.Name, &toAdd.OwnerID)
		if err != nil {
			return toReturn, err
		}
		//Add this result to ToReturn
		toReturn = append(toReturn, toAdd)
	}

	return toReturn, nil
}

//GetRootPages returns incomplete page data for root pages of the specified user (Content not included)
func (DBConnection *MariaDBPlugin) GetRootPages(userID uint64) ([]interfaces.Page, error) {
	var toReturn []interfaces.Page
	if userID == 0 {
		return toReturn, errors.New("User ID not provided")
	}

	query := "SELECT ID, Name FROM Pages WHERE OwnerID=? AND (PrevID IS NULL OR PrevID=0)"

	//Now we have query and args, run the query
	rows, err := DBConnection.DBHandle.Query(query, userID)
	if err != nil {
		return toReturn, err
	}
	defer rows.Close()

	//For each row
	for rows.Next() {
		toAdd := interfaces.Page{PrevID: 0, OwnerID: userID}
		//Parse out the data
		err := rows.Scan(&toAdd.ID, &toAdd.Name)
		if err != nil {
			return toReturn, err
		}
		//Add this result to ToReturn
		toReturn = append(toReturn, toAdd)
	}

	return toReturn, nil
}

//GetPagePath returns the a slice representing the page up to the root
func (DBConnection *MariaDBPlugin) GetPagePath(pageID uint64, rootFirst bool) ([]interfaces.Page, error) {
	var toReturn []interfaces.Page
	if pageID == 0 {
		return toReturn, errors.New("Page ID not provided")
	}

	//Starting with current page, work our way up until we hit the root
	//Add first page
	nextID := pageID

	for nextID != 0 {
		pageData, err := DBConnection.GetPage(nextID)
		if err != nil {
			return toReturn, err
		}
		nextID = pageData.PrevID
		toReturn = append(toReturn, pageData)
	}

	if rootFirst {
		//Invert slice as it is in reverse order of request
		for i, j := 0, len(toReturn)-1; i < j; i, j = i+1, j-1 {
			toReturn[i], toReturn[j] = toReturn[j], toReturn[i]
		}
	}

	return toReturn, nil
}

//SearchPages returns incomplete page data for for pages that match the supplied query
func (DBConnection *MariaDBPlugin) SearchPages(userID uint64, searchquery string, limit uint64, offset uint64) ([]interfaces.Page, error) {
	var toReturn []interfaces.Page
	if userID == 0 {
		return toReturn, errors.New("User ID not provided")
	}
	if searchquery == "" {
		return toReturn, errors.New("Query not provided")
	}
	if limit == 0 {
		return toReturn, errors.New("Limit not provided")
	}

	query := "SELECT ID, Name, Content FROM Pages WHERE OwnerID=? AND (MATCH (Content) AGAINST (? IN BOOLEAN MODE)) LIMIT ? OFFSET ?;"

	//Now we have query and args, run the query
	rows, err := DBConnection.DBHandle.Query(query, userID, searchquery, limit, offset)
	if err != nil {
		return toReturn, err
	}
	defer rows.Close()

	//For each row
	for rows.Next() {
		toAdd := interfaces.Page{PrevID: 0, OwnerID: userID}
		//Parse out the data
		err := rows.Scan(&toAdd.ID, &toAdd.Name, &toAdd.Content)
		if err != nil {
			return toReturn, err
		}
		//Add this result to ToReturn
		toReturn = append(toReturn, toAdd)
	}

	return toReturn, nil
}

//GetPageRevisions returns a slice of page revisions given a pageID
func (DBConnection *MariaDBPlugin) GetPageRevisions(pageID uint64, limit uint64, offset uint64) ([]interfaces.Page, uint64, error) {
	var toReturn []interfaces.Page
	if pageID == 0 {
		return toReturn, 0, errors.New("Page ID not provided")
	}
	if limit == 0 {
		return toReturn, 0, errors.New("Limit not provided")
	}

	MaxCount, err := DBConnection.GetPageRevisionCount(pageID)
	if err != nil {
		return toReturn, MaxCount, errors.New("failed to get count in subquery: " + err.Error())
	}

	query := "SELECT ID, Name, Content, UpdateTime FROM PageRevisions WHERE PageID=? ORDER BY ID DESC LIMIT ? OFFSET ?;"

	//Now we have query and args, run the query
	rows, err := DBConnection.DBHandle.Query(query, pageID, limit, offset)
	if err != nil {
		return toReturn, MaxCount, err
	}
	defer rows.Close()

	//For each row
	for rows.Next() {
		var RevisionTime mysql.NullTime

		toAdd := interfaces.Page{PrevID: 0, ID: pageID}
		//Parse out the data
		err := rows.Scan(&toAdd.RevisionID, &toAdd.Name, &toAdd.Content, &RevisionTime)
		if err != nil {
			return toReturn, MaxCount, err
		}
		if RevisionTime.Valid {
			toAdd.RevisionTime = RevisionTime.Time
		}
		//Add this result to ToReturn
		toReturn = append(toReturn, toAdd)
	}

	return toReturn, MaxCount, nil
}

//GetPageRevisionCount returns a count of page revisions given a pageID
func (DBConnection *MariaDBPlugin) GetPageRevisionCount(pageID uint64) (uint64, error) {
	var ToReturn uint64
	if pageID == 0 {
		return ToReturn, errors.New("Page ID not provided")
	}

	query := "SELECT COUNT(ID) FROM PageRevisions WHERE PageID=?;"

	//Now we have query and args, run the query
	err := DBConnection.DBHandle.QueryRow(query, pageID).Scan(&ToReturn)
	if err != nil {
		return ToReturn, err
	}
	return ToReturn, nil
}

//GetPageRevision returns specific page revision (Incomplete as revisions only contain partial information)
func (DBConnection *MariaDBPlugin) GetPageRevision(pageID uint64, revisionID uint64) (interfaces.Page, error) {
	toReturn := interfaces.Page{ID: pageID, RevisionID: revisionID}
	if pageID == 0 {
		return toReturn, errors.New("Page ID not provided")
	}
	if revisionID == 0 {
		return toReturn, errors.New("Revision ID not provided")
	}

	query := "SELECT Name, Content, UpdateTime FROM PageRevisions WHERE PageID=? AND ID=?;"

	//Now we have query and args, run the query
	var RevisionTime mysql.NullTime
	err := DBConnection.DBHandle.QueryRow(query, pageID, revisionID).Scan(&toReturn.Name, &toReturn.Content, &RevisionTime)
	if RevisionTime.Valid {
		toReturn.RevisionTime = RevisionTime.Time
	}
	return toReturn, err
}
