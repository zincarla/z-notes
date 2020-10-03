package templatecache

import (
	"html/template"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"z-notes/config"
	"z-notes/logging"
)

//TemplateCache contains a cache of templates used by the server
var TemplateCache = template.New("")

//CacheTemplates loads the TemplateCache. This should be called before use
func CacheTemplates() error {
	var allFiles []string
	files, err := ioutil.ReadDir(config.Configuration.HTTPRoot)
	if err != nil {
		logging.WriteLog(logging.LogLevelError, "templatcache/CacheTemplates", "*", logging.ResultFailure, []string{"failed to read template files from httproot", err.Error()})
		return err
	}
	for _, file := range files {
		filename := file.Name()
		if strings.HasSuffix(filename, ".html") {
			allFiles = append(allFiles, path.Join(config.Configuration.HTTPRoot, filename))
		}
	}

	templates := template.New("")

	templates, err = templates.ParseFiles(allFiles...)
	if err != nil {
		logging.WriteLog(logging.LogLevelError, "templatcache/CacheTemplates", "*", logging.ResultFailure, []string{"failed to prase template files from httproot", err.Error()})
		return err
	}
	TemplateCache = templates
	logging.WriteLog(logging.LogLevelInfo, "templatcache/CacheTemplates", "*", logging.ResultInfo, []string{"Added Templates", strconv.Itoa(len(allFiles))})
	return nil
}
