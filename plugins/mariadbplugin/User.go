package mariadbplugin

import (
	"errors"
	"z-notes/interfaces"
	"z-notes/logging"

	"github.com/go-sql-driver/mysql"
)

//CreateUser is used to create and add a user to the AuthN database (return nil on success)
func (DBConnection *MariaDBPlugin) CreateUser(userData interfaces.UserInformation) (uint64, error) {
	//Must have email, username, OIDCIssuer, OIDCSubject
	if userData.OIDCSubject == "" || userData.OIDCIssuer == "" {
		return 0, errors.New("OIDC information not provided")
	}
	if userData.Name == "" {
		return 0, errors.New("username not provided")
	}
	if userData.EMail == "" {
		return 0, errors.New("email not provided")
	}
	if !userData.EMailVerified {
		return 0, errors.New("user email not verfied with oidc provider")
	}
	resultInfo, err := DBConnection.DBHandle.Exec("INSERT INTO Users (Name, EMail, OIDCIssuer, OIDCSubject) VALUES (?, ?, ?, ?);", userData.Name, userData.EMail, userData.OIDCIssuer, userData.OIDCSubject)
	if err != nil {
		return 0, err
	}
	id, err := resultInfo.LastInsertId()
	return uint64(id), err
}

//UpdateUserNameEmail updates a user based on DBID to have the name/email located in the userData object. If an email has not been verified, it will be silently ignored
func (DBConnection *MariaDBPlugin) UpdateUserNameEmail(userData interfaces.UserInformation) error {
	if userData.Name == "" && userData.EMail == "" {
		return errors.New("username and email not provided")
	}
	if userData.DBID == 0 {
		return errors.New("DBID not provided")
	}
	query := "UPDATE Users SET"
	queryArray := []interface{}{}
	if userData.Name != "" {
		query = query + " Name=?"
		queryArray = append(queryArray, userData.Name)
	}
	if userData.EMail != "" && userData.EMailVerified {
		if userData.Name != "" {
			query = query + ", "
		}
		query = query + "EMail=?"
		queryArray = append(queryArray, userData.EMail)
	}
	query = query + " WHERE ID=?"
	logging.WriteLog(logging.LogLevelDebug, "MAriaDB/User/UpdateUserNameEmail", "", logging.ResultInfo, []string{query})
	queryArray = append(queryArray, userData.DBID)
	_, err := DBConnection.DBHandle.Exec(query, queryArray...)
	return err
}

//RemoveUser Removes a user from the user database (nil on success)
func (DBConnection *MariaDBPlugin) RemoveUser(userID uint64) error {
	_, err := DBConnection.DBHandle.Exec("DELETE FROM Users WHERE ID = ?", userID)
	return err
}

//SetUserDisableState disables or enables a user account
func (DBConnection *MariaDBPlugin) SetUserDisableState(userID uint64, isDisabled bool) error {
	_, err := DBConnection.DBHandle.Exec("UPDATE Users SET Disabled=? WHERE ID=?", isDisabled, userID)
	return err
}

//GetUser returns a completed UserInformation object for the user specified, OIDCIssuer and Subject must be specified, or the DBID
func (DBConnection *MariaDBPlugin) GetUser(userData interfaces.UserInformation) (interfaces.UserInformation, error) {
	//Prefer DBID
	query := "SELECT ID, Name, EMail, OIDCIssuer, OIDCSubject, CreationTime, Disabled FROM Users WHERE ID=?"
	queryArray := []interface{}{}
	if userData.DBID != 0 {
		queryArray = append(queryArray, userData.DBID)
	} else if userData.OIDCIssuer != "" && userData.OIDCSubject != "" {
		query = "SELECT ID, Name, EMail, OIDCIssuer, OIDCSubject, CreationTime, Disabled FROM Users WHERE OIDCIssuer=? AND OIDCSubject=?"
		queryArray = append(queryArray, userData.OIDCIssuer)
		queryArray = append(queryArray, userData.OIDCSubject)
	} else if userData.EMail != "" {
		query = "SELECT ID, Name, EMail, OIDCIssuer, OIDCSubject, CreationTime, Disabled FROM Users WHERE EMail=?"
		queryArray = append(queryArray, userData.EMail)
	} else {
		return userData, errors.New("incomplete identity provided, need either DBID, EMail or the OIDC information to pull a user from database")
	}
	var NCreationTime mysql.NullTime
	err := DBConnection.DBHandle.QueryRow(query, queryArray...).Scan(&userData.DBID, &userData.Name, &userData.EMail, &userData.OIDCIssuer, &userData.OIDCSubject, &NCreationTime, &userData.Disabled)
	if err != nil {
		return userData, err
	}
	if NCreationTime.Valid {
		userData.CreationTime = NCreationTime.Time
	}
	return userData, nil
}
