package main

// Import needed packages
import (
	"encoding/json"
	"github.com/ant0ine/go-json-rest/rest"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// Define finspect internal constants (TODO: move config_dir const to a command flag)
const (
	FINSPECT_NAME        string = "finspect"
	FINSPECT_DESCRIPTION string = ""
	FINSPECT_VERSION     string = "0.0.1a"
	FINSPECT_AUTHOR      string = "Patrick Kuti <hello@introspect.in>"
	FINSPECT_CONFIG_DIR  string = "/etc/finspect/conf/"
)

// Define log formats for finspect-http server
const (
	CommonLogFormat   = "%h %l %u %t \"%r\" %s %b"                                                         // CLR
	CombinedLogFormat = "%h %l %u %t \"%r\" %s %b \"%{Referer}i\" \"%{User-Agent}i\""                      // NCSA extended/combined log format
	DefaultLogFormat  = "%t %S\033[0m \033[36;1m%Dμs\033[0m \"%r\" \033[1;30m%u \"%{User-Agent}i\"\033[0m" // Debug Format
)

// JSON struct for finspect-http server configuration
type Configuration struct {
	LogDirectory string
}

// Variable for global finspect-http server configuration
var FinspectHttpConfiguration *Configuration = &Configuration{}

// setDefaults sets default values for the configuration struct
func (self *Configuration) setDefaults() {
	self.LogDirectory = "/var/log/finspect/"
}

// Init parses configuration files, compiles global variables
// sets up logging and in general sets up the finspect-http server
func init() {

	// Check if finspect server configuration file exists
	FinspectHttpConfigurationFilePath := FINSPECT_CONFIG_DIR + "http.json"
	_, err := os.Stat(FinspectHttpConfigurationFilePath)
	if err != nil {
		log.Fatal("Error looking for configuration file: " + FinspectHttpConfigurationFilePath)
		os.Exit(1)
	}

	// Set default configurations
	FinspectHttpConfiguration.setDefaults()

	// Attempt to load and parse json configuration file into global configuration variable
	_loadConfig(FinspectHttpConfigurationFilePath)

	// Check if log directory exists and is a directory
	LogDirectory, err := os.Stat(FinspectHttpConfiguration.LogDirectory)
	if err != nil {
		log.Fatal("Error trying to access log directory ("+FinspectHttpConfiguration.LogDirectory+"), error returned was: ", err)
		os.Exit(1)
	}
	if !LogDirectory.IsDir() {
		log.Fatal("Log directory is not a directory, error returned was: ", err)
		os.Exit(1)
	}
}

// _loadConfig loads values from a configuration file into global a configuration variable
func _loadConfig(FinspectHttpConfigurationFilePath string) {

	// Try to open the finspect-http server configuration file and read it into global variable
	FinspectHttpConfigurationFile, err := ioutil.ReadFile(FinspectHttpConfigurationFilePath)
	if err != nil {
		log.Fatal("Error reading configuration file, error returned was: ", err)
		os.Exit(1)
	}

	// Decode JSON from configuration file and dump into configuration struct to override default values
	err = json.Unmarshal(FinspectHttpConfigurationFile, FinspectHttpConfiguration)
	if err != nil {
		log.Fatal("Error parsing configuration file, please ensure it is valid JSON. Error returned was: ", err)
		os.Exit(1)
	}

}

// Main defines the available REST resources, starts up the finspect-http server
// along with various configuration options, settings and sets up logging
func main() {

	// Default generic log handler
	LogFileHandler, err := os.OpenFile(FinspectHttpConfiguration.LogDirectory+"http.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Error trying to open general logfile, error returned was: ", err)
		os.Exit(1)
	}
	defer LogFileHandler.Close()
	log.SetOutput(LogFileHandler)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// HTTP access log handler
	AccessLogFileHandler, err := os.OpenFile(FinspectHttpConfiguration.LogDirectory+"http-access.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Error trying to open access logfile, error returned was: ", err)
		os.Exit(1)
	}
	defer AccessLogFileHandler.Close()

	// HTTP error log handler
	ErrorLogFileHandler, err := os.OpenFile(FinspectHttpConfiguration.LogDirectory+"http-error.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Error trying to open error logfile, error returned was: ", err)
		os.Exit(1)
	}
	defer ErrorLogFileHandler.Close()

	// Setting up configuration for the HTTP REST router
	handler := rest.ResourceHandler{
		EnableGzip:               true,
		DisableJsonIndent:        false,
		EnableStatusService:      true,
		EnableResponseStackTrace: true,
		EnableLogAsJson:          false,
		EnableRelaxedContentType: false,
		Logger:            log.New(AccessLogFileHandler, "", 0),
		LoggerFormat:      CombinedLogFormat,
		DisableLogger:     false,
		ErrorLogger:       log.New(ErrorLogFileHandler, "", 0),
		XPoweredBy:        FINSPECT_NAME + " - " + FINSPECT_VERSION,
		DisableXPoweredBy: false,
	}

	// Define some HTTP REST resources, log a fatal error if something goes wrong
	err = handler.SetRoutes(

		// Watcher related routes
		&rest.Route{"GET", "/watchpaths", getReturnWatchPaths},
		&rest.Route{"POST", "/watchpaths", postAddWatchPath},
		&rest.Route{"DELETE", "/watchpaths/:watchpathid:", deleteRemoveWatchPath},
		&rest.Route{"GET", "/watchpaths/:watchpathid:", getReturnWatchPath},

		// Indexer related routes
		&rest.Route{"POST", "/indexjobs", postAddIndexJob},
		&rest.Route{"GET", "/indexjobs/:indexjobid:", getReturnIndexJob},
		&rest.Route{"DELETE", "/indexjobs/:indexjobid:", deleteRemoveIndexJob},
		&rest.Route{"POST", "/indexjobs/search", postSearchIndexJobs},

		// Ingester related routes
		&rest.Route{"POST", "/ingestjobs", postAddIngestJob},
		&rest.Route{"GET", "/ingestjobs/:ingestjobid:", getReturnIngestJob},
		&rest.Route{"DELETE", "/ingestjobs/:ingestjobid:", deleteRemoveIngestJob},
		&rest.Route{"POST", "/ingestjobs/search", postSearchIngestJobs},

		// File related routes
		&rest.Route{"POST", "/files", postCreateFile},

		// Admin and internally available resources
		&rest.Route{"GET", "/.status", func(w rest.ResponseWriter, r *rest.Request) { w.WriteJson(handler.GetStatus()) }},
		&rest.Route{"POST", "/shutdown", postAdminShutdownHttpServer},
		&rest.Route{"POST", "/reload", postAdminReloadHttpServer},
		&rest.Route{"POST", "/restart", postAdminRestartHttpServer},
	)
	if err != nil {
		log.Fatal("Error with HTTP router, error returned was: ", err)
		os.Exit(1)
	}

	// Start the finspect-http server
	log.Fatal(http.ListenAndServe(":7070", &handler))
	os.Exit(1)
}

// Mock watcher functions
func getReturnWatchPaths(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}
func postAddWatchPath(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}
func deleteRemoveWatchPath(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}
func getReturnWatchPath(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}

// Mock indexer functions
func postAddIndexJob(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}
func getReturnIndexJob(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}
func deleteRemoveIndexJob(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}
func postSearchIndexJobs(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}

// Mock ingester functions
func postAddIngestJob(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}
func getReturnIngestJob(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}
func deleteRemoveIngestJob(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}
func postSearchIngestJobs(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}

// Mock files functions
func postCreateFile(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}

// Mock admin and internal functions
func postAdminShutdownHttpServer(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}
func postAdminReloadHttpServer(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}
func postAdminRestartHttpServer(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson("OK")
}
