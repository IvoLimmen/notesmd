package main

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var validPath = regexp.MustCompile(`^/(edit|save|view|delete)/([a-zA-Z0-9\s]+)$`)
var special = regexp.MustCompile(`^/(special)/`)

func makeHandler(fn func(http.ResponseWriter, *http.Request, string, Config), config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := special.FindStringSubmatch(r.URL.Path)
		if s == nil {
			m := validPath.FindStringSubmatch(r.URL.Path)
			if m == nil {
				http.NotFound(w, r)
				return
			}
			fn(w, r, m[2], config)
		} else {
			println(s)
			fn(w, r, r.URL.Path, config)
		}

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

func deleteHandler(w http.ResponseWriter, r *http.Request, title string, config Config) {
	ok, err := deletePage(title, config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if ok {
		http.Redirect(w, r, "/view/Index", http.StatusFound)
	}
}

func editHandler(w http.ResponseWriter, r *http.Request, title string, config Config) {
	p, err := loadPage(title, config)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func fileUploadHandler(w http.ResponseWriter, r *http.Request, config Config) {
	r.ParseMultipartForm(10 << 20) // 10MB

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Now let’s save it locally
	dst, err := createFile(handler.Filename, config)
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the destination file
	if _, err := dst.ReadFrom(file); err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
	}

	http.Redirect(w, r, "/special/Attachments", http.StatusFound)
}

func showAttachments(w http.ResponseWriter, config Config) {
	files := listAttachments(config)
	err := templates.ExecuteTemplate(w, "attachments.html", TemplateView{Title: "Attachments", Files: files, Special: true})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func specialHandler(w http.ResponseWriter, r *http.Request, path string, config Config) {
	parts := strings.Split(path[1:], "/")
	command := parts[1]

	switch command {
	case "AllFiles":
		files := listFiles(config)
		showFiles(w, files, "All files", true)
	case "SearchFiles":
		criteria := r.FormValue("search")
		files, completeMatch := search(listFiles(config), criteria)
		caser := cases.Title(language.English)
		if !completeMatch {
			files = append(files, ExistingFile{FileName: caser.String(criteria), Exists: false})
		}
		title := fmt.Sprintf("Files found with '%s'", criteria)
		showFiles(w, files, title, true)
	case "Attachments":
		showAttachments(w, config)
	case "DelAtt":
		dir := filepath.Join(config.DataDir, "att")
		filename := filepath.Join(dir, parts[2])
		os.Remove(filename)
		http.Redirect(w, r, "/special/Attachments", http.StatusFound)
	case "RandomFile":
		files := listFiles(config)
		file := files[rand.IntN(len(files))]
		http.Redirect(w, r, "/view/"+file.FileName, http.StatusFound)
	default:
		http.Redirect(w, r, "/view/Index", http.StatusFound)
	}
}
