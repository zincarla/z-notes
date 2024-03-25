package mariadbplugin

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"io"
	"z-notes/interfaces"

	"github.com/go-sql-driver/mysql"
	"github.com/mr-tron/base58"
)

//CreateFriendlyID is used to create a tokenID
func (DBConnection *MariaDBPlugin) CreateFriendlyID() (string, error) {
	//Create hard to guess FriendlyID
	rawKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, rawKey); err != nil {
		return "", errors.New("token could not be created as and ID could not be generated: " + err.Error())
	}

	return string(base58.Encode(rawKey)), nil
}

//CreateUser is used to create and add a user to the AuthN database (return nil on success)
func (DBConnection *MariaDBPlugin) CreateToken(tokenInfo interfaces.APITokenInformation) (interfaces.APITokenInformation, error) {
	if tokenInfo.OwnerID == 0 {
		return tokenInfo, errors.New("token could not be created as no valid owner provided")
	}
	//Create hard to guess FriendlyID
	FriendlyID, err := DBConnection.CreateFriendlyID()
	if err != nil {
		return tokenInfo, err
	}

	//Add to db
	var resultInfo sql.Result
	if tokenInfo.Expires {
		resultInfo, err = DBConnection.DBHandle.Exec("INSERT INTO APITokens (OwnerID, FriendlyID, ExpireTime) VALUES (?, ?, ?);", tokenInfo.OwnerID, FriendlyID, tokenInfo.ExpirationTime)
		if err != nil {
			return tokenInfo, err
		}
	} else {
		resultInfo, err = DBConnection.DBHandle.Exec("INSERT INTO APITokens (OwnerID, FriendlyID, ExpireTime) VALUES (?, ?, NULL);", tokenInfo.OwnerID, FriendlyID)
		if err != nil {
			return tokenInfo, err
		}
	}
	//Update tokenIfno
	newTokenID, err := resultInfo.LastInsertId()
	tokenInfo.ID = uint64(newTokenID)
	tokenInfo.FriendlyID = FriendlyID
	//Save out
	return tokenInfo, err
}

//GetToken returns a token based on FriendlyID
func (DBConnection *MariaDBPlugin) GetToken(tokenID string) (interfaces.APITokenInformation, error) {
	var tokenInfo interfaces.APITokenInformation
	//Prefer DBID
	query := "SELECT ID, OwnerID, CreationTime, ExpireTime FROM APITokens WHERE FriendlyID=?"
	var NCreationTime mysql.NullTime
	var NExpirationTime mysql.NullTime
	err := DBConnection.DBHandle.QueryRow(query, tokenID).Scan(&tokenInfo.ID, &tokenInfo.OwnerID, &NCreationTime, &NExpirationTime)
	if err != nil {
		return tokenInfo, err
	}
	if NCreationTime.Valid {
		tokenInfo.CreationTime = NCreationTime.Time
	}
	if NExpirationTime.Valid {
		tokenInfo.ExpirationTime = NExpirationTime.Time
		tokenInfo.Expires = true
	}
	tokenInfo.FriendlyID = tokenID
	return tokenInfo, nil
}

//GetTokens returns a slice of tokens based on UserID
func (DBConnection *MariaDBPlugin) GetTokens(userID uint64) ([]interfaces.APITokenInformation, error) {
	var tokenInfo []interfaces.APITokenInformation
	query := "SELECT ID, FriendlyID, CreationTime, ExpireTime FROM APITokens WHERE OwnerID=?"

	rows, err := DBConnection.DBHandle.Query(query, userID)
	if err != nil {
		return tokenInfo, err
	}
	defer rows.Close()

	//For each row
	for rows.Next() {
		var NExpirationTime mysql.NullTime
		var NCreationTime mysql.NullTime

		toAdd := interfaces.APITokenInformation{OwnerID: userID}
		//Parse out the data
		err := rows.Scan(&toAdd.ID, &toAdd.FriendlyID, &NCreationTime, &NExpirationTime)
		if err != nil {
			return tokenInfo, err
		}
		if NCreationTime.Valid {
			toAdd.CreationTime = NCreationTime.Time
		}
		if NExpirationTime.Valid {
			toAdd.ExpirationTime = NExpirationTime.Time
			toAdd.Expires = true
		}
		//Add this result to ToReturn
		tokenInfo = append(tokenInfo, toAdd)
	}
	return tokenInfo, nil
}

//RefreshToken refreshes a token by crating a new tokenFriendlyID returns the new tokenFriendlyID, and/or an error
func (DBConnection *MariaDBPlugin) RefreshToken(tokenInfo interfaces.APITokenInformation) (interfaces.APITokenInformation, error) {
	FriendlyID, err := DBConnection.CreateFriendlyID()
	if err != nil {
		return tokenInfo, err
	}

	query := "UPDATE APITokens SET FriendlyID=?, ExpireTime=? WHERE FriendlyID=?"
	var NExpirationTime sql.NullTime
	if tokenInfo.Expires {
		NExpirationTime = sql.NullTime{Time: tokenInfo.ExpirationTime, Valid: true}
	}
	_, err = DBConnection.DBHandle.Exec(query, FriendlyID, NExpirationTime, tokenInfo.FriendlyID)

	tokenInfo.FriendlyID = FriendlyID
	return tokenInfo, err
}

//RemoveToken deletes a token from the database
func (DBConnection *MariaDBPlugin) RemoveToken(tokenFriendlyID string) error {
	query := "DELETE FROM APITokens WHERE FriendlyID=?"
	_, err := DBConnection.DBHandle.Exec(query, tokenFriendlyID)
	return err
}
