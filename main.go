package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
	"z-notes/config"
	"z-notes/database"
	"z-notes/logging"
	"z-notes/plugins"
	"z-notes/plugins/mariadbplugin"
	"z-notes/routers"
	"z-notes/routers/api"
	"z-notes/routers/templatecache"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

func main() {
	//Load succeeded
	configConfirmed := false
	//Init plugins
	logging.LogInterface = &plugins.STDLog{}
	//Load Configuration
	configPath := "." + string(filepath.Separator) + "configuration" + string(filepath.Separator) + "config.json"
	err := config.LoadConfiguration(configPath)
	if err != nil {
		config.Configuration.TargetLogLevel = 100
		logging.WriteLog(logging.LogLevelWarning, "main/Main", "*", logging.ResultFailure, []string{err.Error(), "Will use/save default file"})
	}

	//Add any missing configs
	fixMissingConfigs()

	//Init logging
	logging.LogInterface.Init(config.Configuration.TargetLogLevel, config.Configuration.LoggingWhiteList, config.Configuration.LoggingBlackList)

	//Resave config file
	config.SaveConfiguration(configPath)

	//Init webserver cache
	templatecache.CacheTemplates()

	//Init markdown parser
	routers.InitMarkdown()

	//If we can, start the database
	//logging.WriteLog("main/Main", "*", "Information", []string{fmt.Sprintf("%+v", config.Configuration)})
	if config.Configuration.DBName == "" || config.Configuration.DBPassword == "" || config.Configuration.DBUser == "" || config.Configuration.DBHost == "" {
		logging.WriteLog(logging.LogLevelCritical, "main/Main", "*", logging.ResultFailure, []string{"Missing database information. (Instance, User, Password?)"})
	} else {
		//Initialize DB Connection
		database.DBInterface = &mariadbplugin.MariaDBPlugin{}
		err = database.DBInterface.InitDatabase()
		if err != nil {
			logging.WriteLog(logging.LogLevelError, "main/Main", "*", logging.ResultFailure, []string{"Failed to connect to database. Will keep trying. ", err.Error()})
			//Wait group for ending server
			serverEndedWG := &sync.WaitGroup{}
			serverEndedWG.Add(1)
			//Setup basic routers and server server
			requestRouter := mux.NewRouter()
			requestRouter.HandleFunc("/", routers.BadConfigRouter)
			requestRouter.HandleFunc("/resources/{file}", routers.ResourceRouter) //Required for CSS
			server := &http.Server{
				Handler:        requestRouter,
				Addr:           config.Configuration.Address,
				ReadTimeout:    config.Configuration.ReadTimeout,
				WriteTimeout:   config.Configuration.WriteTimeout,
				MaxHeaderBytes: config.Configuration.MaxHeaderBytes,
			}
			//Actually start server listener in a goroutine
			go badConfigServerListenAndServe(serverEndedWG, server)
			//Now we loop for database connection
			for err != nil {
				time.Sleep(60 * time.Second) // retry interval
				err = database.DBInterface.InitDatabase()
			}
			//Kill server once we get a database connection
			waitCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			//Not defering cancel as this is the main function, instead calling it below after it is uneeded
			if err := server.Shutdown(waitCtx); err != nil {
				logging.WriteLog(logging.LogLevelError, "main/Main", "*", logging.ResultFailure, []string{"Error shutting down temp server. ", err.Error()})
			}
			cancel()
		}
		logging.WriteLog(logging.LogLevelInfo, "main/Main", "*", logging.ResultSuccess, []string{"Successfully connected to database"})
		configConfirmed = true
	}

	//Verify OpenID
	if config.Configuration.OpenIDClientID == "" || config.Configuration.OpenIDCallbackURL == "" || config.Configuration.OpenIDEndpointURL == "" {
		configConfirmed = false
		logging.WriteLog(logging.LogLevelCritical, "main/Main", "*", logging.ResultFailure, []string{"OpenID config missing"})
	} else {
		err = routers.InitOAuth()
		if err != nil {
			configConfirmed = false
			logging.WriteLog(logging.LogLevelCritical, "main/Main", "*", logging.ResultFailure, []string{"OpenID failed to initialize", err.Error()})
		}
	}

	//Verify TLS Settings
	if config.Configuration.UseTLS {
		if _, err := os.Stat(config.Configuration.TLSCertPath); err != nil {
			configConfirmed = false
			logging.WriteLog(logging.LogLevelCritical, "main/Main", "*", logging.ResultFailure, []string{"Failed to stat TLS Cert file, does it exist? Does this application have permission to it?"})
		} else if _, err := os.Stat(config.Configuration.TLSKeyPath); err != nil {
			configConfirmed = false
			logging.WriteLog(logging.LogLevelCritical, "main/Main", "*", logging.ResultFailure, []string{"Failed to stat TLS Key file, does it exist? Does this application have permission to it?"})
		}
	}
	//Setup request routers
	requestRouter := mux.NewRouter()

	//Add router paths
	if configConfirmed == true {
		//Web routers
		requestRouter.HandleFunc("/resources/{file}", routers.ResourceRouter).Methods("GET")
		requestRouter.HandleFunc("/", routers.RootRouter).Methods("GET")
		requestRouter.HandleFunc("/about/{file}", routers.AboutRouter).Methods("GET")
		//requestRouter.HandleFunc("/redirect", routers.RedirectRouter).Methods("GET")
		requestRouter.HandleFunc("/openidc/logon", routers.AuthRouter).Methods("GET")
		requestRouter.HandleFunc("/openidc/callback", routers.AuthCallback).Methods("GET")
		requestRouter.HandleFunc("/openidc/logout", routers.AuthLogoutRouter).Methods("GET")

		requestRouter.HandleFunc("/page/{pageID}/view", routers.PageRouter).Methods("GET")
		requestRouter.HandleFunc("/page/{pageID}/resources/{resource}", routers.PageResourceRouter).Methods("GET")
		requestRouter.HandleFunc("/createpage", routers.CreatePageRouter).Methods("POST")
		requestRouter.HandleFunc("/search", routers.SearchRouter).Methods("GET")
		requestRouter.HandleFunc("/page/{pageID}/edit", routers.EditPageGetRouter).Methods("GET")
		requestRouter.HandleFunc("/page/{pageID}/edit", routers.EditPagePostRouter).Methods("POST")
		requestRouter.HandleFunc("/page/{pageID}/file/upload", routers.UploadFilePostRouter).Methods("POST")
		requestRouter.HandleFunc("/page/{pageID}/file", routers.UploadFileGetRouter).Methods("GET")
		requestRouter.HandleFunc("/page/{pageID}/file/delete", routers.DeleteFilePostRouter).Methods("POST")
		requestRouter.HandleFunc("/page/{pageID}/security", routers.SecurityPageGetRouter).Methods("GET")
		requestRouter.HandleFunc("/page/{pageID}/security/add", routers.SecurityPagePostRouter).Methods("POST")
		requestRouter.HandleFunc("/page/{pageID}/security/delete", routers.SecurityPageDeletePostRouter).Methods("POST")
		requestRouter.HandleFunc("/page/{pageID}/move", routers.MovePageGetRouter).Methods("GET")
		requestRouter.HandleFunc("/page/{pageID}/move", routers.MovePagePostRouter).Methods("POST")
		//requestRouter.HandleFunc("/mod", routers.ModRouter)
		//requestRouter.HandleFunc("/mod/user", routers.ModUserRouter)

		//API routers
		requestRouter.HandleFunc("/api/notes/{pageID}/children", api.NoteChildrenGetAPIRouter).Methods("GET")
		//requestRouter.HandleFunc("/api/Logout", api.LogoutAPIRouter)
		//requestRouter.HandleFunc("/api/Users", api.UsersAPIRouter)

	} else {
		requestRouter.HandleFunc("/", routers.BadConfigRouter)
		requestRouter.HandleFunc("/resources/{file}", routers.ResourceRouter) /*Required for CSS*/
	}

	requestRouter.Use(routers.LogMiddleware)

	//Setup csrf protected routers
	csrfRequestRouter := csrf.Protect(config.Configuration.CSRFKey, csrf.RequestHeader("Authenticity-Token"))(requestRouter)

	//Create server
	server := &http.Server{
		Handler:        csrfRequestRouter,
		Addr:           config.Configuration.Address,
		ReadTimeout:    config.Configuration.ReadTimeout,
		WriteTimeout:   config.Configuration.WriteTimeout,
		MaxHeaderBytes: config.Configuration.MaxHeaderBytes,
	}
	//Serve requests. Log on failure.
	logging.WriteLog(logging.LogLevelInfo, "main/Main", "*", logging.ResultSuccess, []string{"Server now listening"})
	if config.Configuration.UseTLS == false || configConfirmed == false {
		err = server.ListenAndServe()
	} else {
		logging.WriteLog(logging.LogLevelInfo, "main/Main", "*", logging.ResultSuccess, []string{"via tls"})
		err = server.ListenAndServeTLS(config.Configuration.TLSCertPath, config.Configuration.TLSKeyPath)
	}
	if err != nil {
		logging.WriteLog(logging.LogLevelCritical, "main/Main", "*", logging.ResultFailure, []string{err.Error()})
	}
}

func fixMissingConfigs() {
	if config.Configuration.Address == "" {
		config.Configuration.Address = ":8080"
	}
	if config.Configuration.PageDirectory == "" {
		config.Configuration.PageDirectory = "." + string(filepath.Separator) + "files"
	}
	if config.Configuration.HTTPRoot == "" {
		config.Configuration.HTTPRoot = "." + string(filepath.Separator) + "http"
	}
	if config.Configuration.MaxUploadBytes <= 0 {
		config.Configuration.MaxUploadBytes = 100 << 20
	}
	if config.Configuration.MaxHeaderBytes <= 0 {
		config.Configuration.MaxHeaderBytes = 1 << 20
	}
	if config.Configuration.ReadTimeout.Nanoseconds() <= 0 {
		config.Configuration.ReadTimeout = 30 * time.Second
	}
	if config.Configuration.WriteTimeout.Nanoseconds() <= 0 {
		config.Configuration.WriteTimeout = 30 * time.Second
	}
	if config.Configuration.OpenIDLogonExpireTime == 0 {
		config.Configuration.OpenIDLogonExpireTime = 1209600
	}
	if config.Configuration.MaxQueryResults == 0 {
		config.Configuration.MaxQueryResults = 20
	}
	if config.Configuration.OpenIDCallbackURL == "" {
		config.Configuration.OpenIDCallbackURL = "http://localhost:8080/openidc/callback"
	}
	if config.Configuration.MaxEmbedSize == 0 {
		config.Configuration.MaxEmbedSize = config.Configuration.MaxUploadBytes
	}
	config.CreateSessionStore()
}

func badConfigServerListenAndServe(serverEndedWG *sync.WaitGroup, server *http.Server) {
	defer serverEndedWG.Done()
	logging.WriteLog(logging.LogLevelInfo, "main/badConfigServerListenAndServe", "*", logging.ResultSuccess, []string{"Temp server now listening"})
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logging.WriteLog(logging.LogLevelCritical, "main/badConfigServerListenAndServe", "*", logging.ResultFailure, []string{"Error occured on temp server stop", err.Error()})
	}
}
