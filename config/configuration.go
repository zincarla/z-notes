package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

//ConfigurationSettings contains the structure of all the settings that will be loaded at runtime.
type ConfigurationSettings struct {
	//DBName is the name of the db used for this instance
	DBName string
	//DBUser is the user name used to auth to the db
	DBUser string
	//DBPassword is the password used to auth to the db
	DBPassword string
	//DBPort the port the database is listening to
	DBPort string
	//DBHost hostname of the database server
	DBHost string
	//PageDirectory path to where page files are stored
	PageDirectory string
	//Address hostname/port that this server should listen on
	Address string
	//ReadTimeout timeout allowed for reads
	ReadTimeout time.Duration
	//WriteTimeout timeout allowed for writes
	WriteTimeout time.Duration
	//MaxHeaderBytes maximum amount of bytes allowed in a request header
	MaxHeaderBytes int
	//SessionStoreKey stores the key to the session store
	SessionStoreKey [][]byte
	//CSRFKey stores the master key for CSRF token
	CSRFKey []byte
	//HTTPRoot directory where template and html files are kept
	HTTPRoot string
	//MaxUploadBytes maximum allowed bytes for an upload
	MaxUploadBytes int64
	//AllowAccountCreation if true, new accounts may be registered
	AllowAccountCreation bool
	//APIThrottle How much time, in milliseconds, users using the API must wait between requests
	APIThrottle int64
	//UseTLS Enables TLS encryption on server
	UseTLS bool
	//TLSCertPath The path to the TLS/SSL cert
	TLSCertPath string
	//TLSKeyPath The path to the TLS/SSL key file for the cert
	TLSKeyPath string
	//OpenIDClientID Client ID for OpenID
	OpenIDClientID string
	//OpenIDClientSecret Client Secrety for OpenID if required
	OpenIDClientSecret string
	//OpenIDCallbackURL is the public address a user should be redirected to after OpenID auth
	OpenIDCallbackURL string
	//OpenIDEndpointURL is the providers endpoint
	OpenIDEndpointURL string
	//OpenIDLogonExpireTime is the time in seconds that a logon should be expired at, forcing relogin. 0=infinite
	OpenIDLogonExpireTime int64
	//TargetLogLevel increase or decrease log verbosity
	TargetLogLevel int64
	//LoggingWhiteList regex based white-list for logging
	LoggingWhiteList string
	//LoggingWhiteList regex based white-list for logging
	LoggingBlackList string
	//MaxQueryResults is how many results to return for a page search, defaults to 20
	MaxQueryResults uint64
}

//SessionStore contains cookie information
var SessionStore *sessions.CookieStore

//Configuration contains all the information loaded from the config file.
var Configuration ConfigurationSettings

//ApplicationVersion Current version of application. This should be incremented every release
var ApplicationVersion = "0.0.0.2"

//SessionVariableName is used when checking cookies
var SessionVariableName = "znotes-session"

//LoadConfiguration loads the specified configuration file into Configuration
func LoadConfiguration(Path string) error {
	//Open the specified file
	File, err := os.Open(Path)
	if err != nil {
		return err
	}
	defer File.Close()
	//Init a JSON Decoder
	decoder := json.NewDecoder(File)
	//Use decoder to decode into a ConfigrationSettings struct
	err = decoder.Decode(&Configuration)
	if err != nil {
		return err
	}
	return nil
}

//SaveConfiguration saves the configuration data in Configuration to the specified file path
func SaveConfiguration(Path string) error {
	//Open the specified file at Path
	File, err := os.OpenFile(Path, os.O_CREATE|os.O_RDWR, 0660)
	defer File.Close()
	if err != nil {
		return err
	}
	//Initialize an encoder to the File
	encoder := json.NewEncoder(File)
	//Encode the settings stored in configuration to File
	err = encoder.Encode(&Configuration)
	if err != nil {
		return err
	}
	return nil
}

//CreateSessionStore will create a new key store given a byte slice. If the slice is nil, a random key will be used.
func CreateSessionStore() {
	if Configuration.SessionStoreKey == nil || len(Configuration.SessionStoreKey) < 2 {
		Configuration.SessionStoreKey = [][]byte{securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32)}
	} else if len(Configuration.SessionStoreKey[0]) != 64 || len(Configuration.SessionStoreKey[1]) != 32 {
		Configuration.SessionStoreKey = [][]byte{securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32)}
	}
	if Configuration.CSRFKey == nil || len(Configuration.CSRFKey) != 32 {
		Configuration.CSRFKey = securecookie.GenerateRandomKey(32)
	}
	SessionStore = sessions.NewCookieStore(Configuration.SessionStoreKey...)
}
