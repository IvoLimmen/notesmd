package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/microcosm-cc/bluemonday"
	"gopkg.in/ini.v1"
)

type Page struct {
	Title string
	Body  template.HTML
	Raw   []byte
}

type Config struct {
	DataDir  string
	AllFiles []string
}

var validPath = regexp.MustCompile("^/(edit|save|view|special)/([a-zA-Z0-9]+)$")
var templates = template.Must(template.ParseFiles("web/templates/edit.html", "web/templates/view.html", "web/templates/allfiles.html"))

func (p *Page) save(config Config) error {
	filename := filepath.Join(config.DataDir, p.Title+".md")
	return os.WriteFile(filename, p.Raw, 0600)
}

func loadPage(title string, config Config) (*Page, error) {
	filename := filepath.Join(config.DataDir, title+".md")
	raw, err := os.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	html := markdown.ToHTML(raw, nil, nil)
	sanitized := bluemonday.UGCPolicy().SanitizeBytes(html)
	body := template.HTML(sanitized)

	return &Page{Title: title, Body: body, Raw: raw}, nil
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
	p := &Page{Title: title, Raw: []byte(body)}
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

func specialHandler(w http.ResponseWriter, r *http.Request, title string, config Config) {
	switch title {
	case "AllFiles":
		files := listFiles(config)
		err := templates.ExecuteTemplate(w, "allfiles.html", files)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case "RandomFile":
		files := listFiles(config)
		file := files[rand.IntN(len(files))]
		http.Redirect(w, r, "/view/"+file, http.StatusFound)
	default:
		http.Redirect(w, r, "/view/Index", http.StatusFound)
	}
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

func listFiles(config Config) []string {
	entries, err := os.ReadDir(config.DataDir)
	if err != nil {
		log.Fatal(err)
	}

	var files []string
	for _, e := range entries {
		base := strings.Split(e.Name(), ".")[0]
		files = append(files, base)
	}

	return files
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
	http.HandleFunc("/special/", makeHandler(specialHandler, config))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/view/Index", http.StatusFound)
	})

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
}
