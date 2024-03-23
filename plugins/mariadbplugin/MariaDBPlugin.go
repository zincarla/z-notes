package mariadbplugin

import (
	"database/sql"
	"errors"
	"strconv"
	"z-notes/config"
	"z-notes/logging"

	"math/rand"
	"time"

	//I mean, where else would this go?
	_ "github.com/go-sql-driver/mysql"
)

//TODO: Increment this whenever we alter the DB Schema, ensure you attempt to add update code below
var currentDBVersion int64 = 2

//TODO: Increment this when we alter the db schema and don't add update code to compensate
var minSupportedDBVersion int64 // 0 by default

//MariaDBPlugin acts as plugin between gib and a Maria/MySQL DB
type MariaDBPlugin struct {
	DBHandle *sql.DB
}

//InitDatabase connects to a database, and if needed, creates and or updates tables
func (DBConnection *MariaDBPlugin) InitDatabase() error {
	rand.Seed(time.Now().UnixNano())
	var err error
	//https://github.com/go-sql-driver/mysql/#dsn-data-source-name
	DBConnection.DBHandle, err = sql.Open("mysql", config.Configuration.DBUser+":"+config.Configuration.DBPassword+"@tcp("+config.Configuration.DBHost+":"+config.Configuration.DBPort+")/"+config.Configuration.DBName)
	if err == nil {
		err = DBConnection.DBHandle.Ping() //Ping actually validates we can query database
		if err == nil {
			version, err := DBConnection.getDatabaseVersion()
			if err == nil {
				logging.WriteLog(logging.LogLevelInfo, "MariaDBPlugin/InitDatabase", "*", logging.ResultInfo, []string{"DBVersion is " + strconv.FormatInt(version, 10)})
				if version < minSupportedDBVersion {
					return errors.New("database version is not supported and no update code was found to bring database up to current version")
				} else if version < currentDBVersion {
					version, err = DBConnection.upgradeDatabase(version)
					if err != nil {
						return err
					}
				}
			} else {
				logging.WriteLog(logging.LogLevelWarning, "MariaDBPlugin/InitDatabase", "*", logging.ResultInfo, []string{"Failed to get database version, assuming not installed. Will attempt to perform install.", err.Error()})
				//Assume no database installed. Perform fresh install
				return DBConnection.performFreshDBInstall()
			}
		}
	}

	return err
}

func (DBConnection *MariaDBPlugin) getDatabaseVersion() (int64, error) {
	var version int64
	row := DBConnection.DBHandle.QueryRow("SELECT version FROM DBVersion")
	err := row.Scan(&version)
	return version, err
}

//performFreshDBInstall Installs the necessary tables for the application. This assumes that the database has not been created before
func (DBConnection *MariaDBPlugin) performFreshDBInstall() error {
	//DBVersion
	_, err := DBConnection.DBHandle.Exec("CREATE TABLE DBVersion (version BIGINT UNSIGNED NOT NULL);")
	if err != nil {
		logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/performFreshDBInstall", "*", logging.ResultFailure, []string{"Failed to install database", err.Error()})
		return err
	}
	_, err = DBConnection.DBHandle.Exec("INSERT INTO DBVersion (version) VALUES (?);", currentDBVersion)
	if err != nil {
		logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/performFreshDBInstall", "*", logging.ResultFailure, []string{"Failed to install database", err.Error()})
		return err
	}
	//Users
	_, err = DBConnection.DBHandle.Exec("CREATE TABLE Users (ID BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE, Name VARCHAR(255) NOT NULL DEFAULT 'User', OIDCIssuer VARCHAR(769) NOT NULL, OIDCSubject VARCHAR(255) NOT NULL, UNIQUE INDEX OIDCIssuerSubject (OIDCIssuer,OIDCSubject), EMail VARCHAR(255) NOT NULL UNIQUE, CreationTime TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL, Disabled BOOL NOT NULL DEFAULT FALSE);")
	if err != nil {
		logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/performFreshDBInstall", "*", logging.ResultFailure, []string{"Failed to install user table in database", err.Error()})
		return err
	}
	//Reserve a couple ids for dynamic permissions
	_, err = DBConnection.DBHandle.Exec("INSERT INTO Users (Name, EMail, OIDCIssuer, OIDCSubject, Disabled) VALUES (?, ?, ?, ?, ?);", "Anonymous", "anonymous@local", "http://local.example/", "anonymous", true)
	if err != nil {
		logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/performFreshDBInstall", "*", logging.ResultFailure, []string{"Failed to install database", err.Error()})
		return err
	}
	_, err = DBConnection.DBHandle.Exec("INSERT INTO Users (Name, EMail, OIDCIssuer, OIDCSubject, Disabled) VALUES (?, ?, ?, ?, ?);", "Authenticated", "authenticated@local", "http://local.example/", "authenticated", true)
	if err != nil {
		logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/performFreshDBInstall", "*", logging.ResultFailure, []string{"Failed to install database", err.Error()})
		return err
	}
	//Pages
	_, err = DBConnection.DBHandle.Exec("CREATE TABLE Pages (ID BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE, PrevID BIGINT UNSIGNED, CONSTRAINT fk_PagesPrevID FOREIGN KEY (PrevID) REFERENCES Pages(ID) ON DELETE CASCADE, INDEX(PrevID), Name VARCHAR(255) NOT NULL DEFAULT 'Unnamed Note', INDEX(Name), OwnerID BIGINT UNSIGNED NOT NULL, INDEX(OwnerID), CONSTRAINT fk_PagesOwnerID FOREIGN KEY (OwnerID) REFERENCES Users(ID) ON DELETE CASCADE, Content MEDIUMTEXT NOT NULL DEFAULT '', FULLTEXT ft_Content (Content));")
	if err != nil {
		logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/performFreshDBInstall", "*", logging.ResultFailure, []string{"Failed to install database", err.Error()})
		return err
	}
	_, err = DBConnection.DBHandle.Exec("CREATE TABLE PageRevisions (ID BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE, UpdateTime TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL, PageID BIGINT UNSIGNED NOT NULL, CONSTRAINT fk_PageRevisionsPageID FOREIGN KEY (PageID) REFERENCES Pages(ID) ON DELETE CASCADE, INDEX(PageID), Name VARCHAR(255) NOT NULL DEFAULT 'Unnamed Note', INDEX(Name), Content MEDIUMTEXT NOT NULL DEFAULT '', FULLTEXT ft_Content (Content));")
	if err != nil {
		logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/performFreshDBInstall", "*", logging.ResultFailure, []string{"Failed to install database", err.Error()})
		return err
	}
	//PagePermissions
	_, err = DBConnection.DBHandle.Exec("CREATE TABLE PagePermissions (ID BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE, PageID BIGINT UNSIGNED NOT NULL, INDEX(PageID), CONSTRAINT fk_PagePermissionsPageID FOREIGN KEY (PageID) REFERENCES Pages(ID) ON DELETE CASCADE, UserID BIGINT UNSIGNED NOT NULL, INDEX(UserID), CONSTRAINT fk_PagePermissionsUserID FOREIGN KEY (UserID) REFERENCES Users(ID) ON DELETE CASCADE, UNIQUE INDEX PageUserPair (PageID,UserID), Permissions BIGINT UNSIGNED NOT NULL DEFAULT 0);")
	if err != nil {
		logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/performFreshDBInstall", "*", logging.ResultFailure, []string{"Failed to install database", err.Error()})
		return err
	}
	//Triggers
	_, err = DBConnection.DBHandle.Exec(`CREATE TRIGGER IF NOT EXISTS CreateRevisionOnUpdate BEFORE UPDATE ON Pages
	FOR EACH ROW
	BEGIN
		INSERT INTO PageRevisions (PageID, Name, Content)
		VALUES (OLD.ID, OLD.Name, OLD.Content);
	END`)
	if err != nil {
		logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/performFreshDBInstall", "*", logging.ResultFailure, []string{"Failed to install database", err.Error()})
		return err
	}
	return nil
}

//TODO: Add update code here
func (DBConnection *MariaDBPlugin) upgradeDatabase(version int64) (int64, error) {
	//Update version 0 -> 1
	if version == 0 {
		//Drop constraints
		_, err := DBConnection.DBHandle.Exec("ALTER TABLE Pages DROP FOREIGN KEY fk_PagesPrevID;")
		if err != nil {
			logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/upgradeDatabase", "*", logging.ResultFailure, []string{"Failed to update database", err.Error()})
			return version, err
		}
		_, err = DBConnection.DBHandle.Exec("ALTER TABLE Pages DROP FOREIGN KEY fk_PagesOwnerID;")
		if err != nil {
			logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/upgradeDatabase", "*", logging.ResultFailure, []string{"Failed to update database", err.Error()})
			return version, err
		}
		_, err = DBConnection.DBHandle.Exec("ALTER TABLE PagePermissions DROP FOREIGN KEY fk_PagePermissionsPageID;")
		if err != nil {
			logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/upgradeDatabase", "*", logging.ResultFailure, []string{"Failed to update database", err.Error()})
			return version, err
		}
		_, err = DBConnection.DBHandle.Exec("ALTER TABLE PagePermissions DROP FOREIGN KEY fk_PagePermissionsUserID;")
		if err != nil {
			logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/upgradeDatabase", "*", logging.ResultFailure, []string{"Failed to update database", err.Error()})
			return version, err
		}
		//Then re-add them
		_, err = DBConnection.DBHandle.Exec("ALTER TABLE Pages ADD CONSTRAINT fk_PagesPrevID FOREIGN KEY (PrevID) REFERENCES Pages(ID) ON DELETE CASCADE;")
		if err != nil {
			logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/upgradeDatabase", "*", logging.ResultFailure, []string{"Failed to update database", err.Error()})
			return version, err
		}
		_, err = DBConnection.DBHandle.Exec("ALTER TABLE Pages ADD CONSTRAINT fk_PagesOwnerID FOREIGN KEY (OwnerID) REFERENCES Users(ID) ON DELETE CASCADE;")
		if err != nil {
			logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/upgradeDatabase", "*", logging.ResultFailure, []string{"Failed to update database", err.Error()})
			return version, err
		}
		_, err = DBConnection.DBHandle.Exec("ALTER TABLE PagePermissions ADD CONSTRAINT fk_PagePermissionsPageID FOREIGN KEY (PageID) REFERENCES Pages(ID) ON DELETE CASCADE;")
		if err != nil {
			logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/upgradeDatabase", "*", logging.ResultFailure, []string{"Failed to update database", err.Error()})
			return version, err
		}
		_, err = DBConnection.DBHandle.Exec("ALTER TABLE PagePermissions ADD CONSTRAINT fk_PagePermissionsUserID FOREIGN KEY (UserID) REFERENCES Users(ID) ON DELETE CASCADE;")
		if err != nil {
			logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/upgradeDatabase", "*", logging.ResultFailure, []string{"Failed to update database", err.Error()})
			return version, err
		}
		//
		_, err = DBConnection.DBHandle.Exec("UPDATE DBVersion SET version = 1;")
		if err != nil {
			logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/upgradeDatabase", "*", logging.ResultFailure, []string{"Failed to update database version", err.Error()})
			return version, err
		}
		version = 1
		logging.WriteLog(logging.LogLevelInfo, "MariaDBPlugin/upgradeDatabase", "*", logging.ResultSuccess, []string{"Database schema updated to version", strconv.FormatInt(version, 10)})
	}
	if version == 1 {
		_, err := DBConnection.DBHandle.Exec("CREATE TABLE PageRevisions (ID BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE, UpdateTime TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL, PageID BIGINT UNSIGNED NOT NULL, CONSTRAINT fk_PageRevisionsPageID FOREIGN KEY (PageID) REFERENCES Pages(ID) ON DELETE CASCADE, INDEX(PageID), Name VARCHAR(255) NOT NULL DEFAULT 'Unnamed Note', INDEX(Name), Content MEDIUMTEXT NOT NULL DEFAULT '', FULLTEXT ft_Content (Content));")
		if err != nil {
			logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/performFreshDBInstall", "*", logging.ResultFailure, []string{"Failed to install database", err.Error()})
			return version, err
		}
		//
		_, err = DBConnection.DBHandle.Exec("UPDATE DBVersion SET version = 2;")
		if err != nil {
			logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/upgradeDatabase", "*", logging.ResultFailure, []string{"Failed to update database version", err.Error()})
			return version, err
		}
		_, err = DBConnection.DBHandle.Exec(`CREATE TRIGGER IF NOT EXISTS CreateRevisionOnUpdate BEFORE UPDATE ON Pages
		FOR EACH ROW
		BEGIN
			INSERT INTO PageRevisions (PageID, Name, Content)
			VALUES (OLD.ID, OLD.Name, OLD.Content);
		END`)
		if err != nil {
			logging.WriteLog(logging.LogLevelCritical, "MariaDBPlugin/performFreshDBInstall", "*", logging.ResultFailure, []string{"Failed to install database", err.Error()})
			return version, err
		}
		version = 2
		logging.WriteLog(logging.LogLevelInfo, "MariaDBPlugin/upgradeDatabase", "*", logging.ResultSuccess, []string{"Database schema updated to version", strconv.FormatInt(version, 10)})
	}
	return version, nil
}
