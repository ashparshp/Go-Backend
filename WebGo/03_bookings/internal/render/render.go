package render

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/ashparshp/bookings/internal/config"
	"github.com/ashparshp/bookings/internal/models"
	"github.com/justinas/nosurf"
)

/*
// RenderTemplate Simply renders a template
func RenderTemplate(w http.ResponseWriter, tmpl string) {
	parsedTemplate, _ := template.ParseFiles("./templates/" + tmpl, "./templates/base.layout.tmpl")
	err := parsedTemplate.Execute(w, nil)

	if err != nil {
		fmt.Println("error parsing template", err)
		return
	}
}
*/

/*
// RenderTemplate renders a template with cache
var tc = make(map[string]*template.Template)
func RenderTemplate(w http.ResponseWriter, t string) {
	var tmpl *template.Template
	var err error

	// check if we already having a template in our cache
	_, inMap := tc[t]

	if !inMap {
		// need to create the template
		log.Println("creating template and adding to cache")
		err = createTemplateCache(t)
		if err != nil {
			log.Println(err)
		}
	} else {
		// we have the template in the cache
		log.Println("using the cached template")
	}

	tmpl = tc[t]
	err = tmpl.Execute(w, nil)
	if err != nil {
		log.Println(err)
	}
}

func createTemplateCache(t string) error {
	templates := []string{
		fmt.Sprintf("./templates/%s", t),"./templates/base.layout.tmpl",
	}

	// parse the template
	tmpl, err := template.ParseFiles(templates...)
	if err != nil {
		return err
	}

	// add template to cache (map)
	tc[t] = tmpl
	return nil
}
*/

// functions to be used in templates
var functions = template.FuncMap{
	"humanDate": HumanDate,
	"formatDate": FormatDate,
	"iterate": Iterate,
	"add": Add,
}

var app *config.AppConfig
var pathToTemplates = "./templates"

func Add(a, b int) int {
	return a + b
}
// Iterate is a helper function to iterate over a number of items
func Iterate(count int) []int {
	var i int
	var items []int
	for i = 0; i < count; i++ {
		items = append(items, i)
	}
	return items
}

// NewRendrer sets the config for the template package
func NewRenderer(a *config.AppConfig) {
	app = a
}

// HumanDate returns time in "YYYY-MM-DD" format
func HumanDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func FormatDate(t time.Time, f string) string {
	return t.Format(f)
}

func AddDefaultData(td *models.TemplateData, r *http.Request) *models.TemplateData {
	td.Flash = app.Session.PopString(r.Context(), "flash")
	td.Error = app.Session.PopString(r.Context(), "error")
	td.Warning = app.Session.PopString(r.Context(), "warning")
	td.CSRFToken = nosurf.Token(r)
	if app.Session.Exists(r.Context(), "user_id") {
		td.IsAuthenticated = 1
	}
	return td
}

// Template renders a template
func Template(w http.ResponseWriter, r *http.Request, tmpl string, td *models.TemplateData) error {
	
	var tc map[string]*template.Template
	if app.UseCahce {
		// get the template cache from the app config
		tc = app.TemplateCache
	} else {
		tc, _ = CreateTemplateCache()
	}

	// get requested template from cache
	t, ok := tc[tmpl]
	if !ok {
		return fmt.Errorf("could not get template from template cache")
	}

	buf := new(bytes.Buffer)

	td = AddDefaultData(td, r)

	err := t.Execute(buf, td)
	if err != nil {
		log.Println(err)
	}

	// render the template
	_, err = buf.WriteTo(w)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("could not write template to response writer")
	}
	return nil
}

func CreateTemplateCache() (map[string]*template.Template, error) {
	myCache := map[string]*template.Template{}

	// get all the files named *.page.tmpl from ./templates
	pages, err := filepath.Glob(fmt.Sprintf("%s/*.page.tmpl", pathToTemplates))

	if err != nil {
		return myCache, err
	}

	// range through all files with *.page.tmpl
	for _, page := range pages {
		name := filepath.Base(page)
		ts, err := template.New(name).Funcs(functions).ParseFiles(page)
		if err != nil {
			return myCache, err
		}
		
		matches, err := filepath.Glob(fmt.Sprintf("%s/*.layout.tmpl", pathToTemplates))
		if err != nil {
			return myCache, err
		}

		if len(matches) > 0 {
			ts, err = ts.ParseGlob(fmt.Sprintf("%s/*.layout.tmpl", pathToTemplates))
			if err != nil {
				return myCache, err
			}
		}

		myCache[name] = ts
	}
	return myCache, nil
}