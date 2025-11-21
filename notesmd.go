package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"gopkg.in/ini.v1"
)

type Page struct {
	Title string
	Body  []byte
}

type Config struct {
	DataDir string
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
var templates = template.Must(template.ParseFiles("web/templates/edit.html", "web/templates/view.html", "web/templates/index.html"))

func (p *Page) save(config Config) error {
	filename := filepath.Join(config.DataDir, p.Title+".txt")
	return os.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string, config Config) (*Page, error) {
	filename := filepath.Join(config.DataDir, title+".txt")
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string, config Config) {
	p, err := loadPage(title, config)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string, config Config) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string, config Config) {
	p, err := loadPage(title, config)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string, Config), config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2], config)
	}
}

func main() {
	cfg, err := ini.Load("notesmd.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	config := Config{DataDir: "undefined"}
	config.DataDir = cfg.Section("paths").Key("data_dir").String()

	port := cfg.Section("server").Key("http_port").MustInt(8080)

	http.HandleFunc("/view/", makeHandler(viewHandler, config))
	http.HandleFunc("/edit/", makeHandler(editHandler, config))
	http.HandleFunc("/save/", makeHandler(saveHandler, config))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "index", &Page{Title: "Index", Body: nil})
	})

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
}
