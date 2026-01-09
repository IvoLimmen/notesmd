package handlers

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ivolimmen/notesmd/file"
	"github.com/ivolimmen/notesmd/types"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var validPath = regexp.MustCompile(`^/(edit|save|view|delete)/([a-zA-Z0-9\s]+)$`)
var special = regexp.MustCompile(`^/(special)/`)

func MakeHandler(fn func(http.ResponseWriter, *http.Request, string, types.Config), config types.Config) http.HandlerFunc {
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

func ViewHandler(w http.ResponseWriter, r *http.Request, title string, config types.Config) {
	p, err := file.LoadPage(title, config)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	file.RenderTemplate(w, "view", p)
}

func SaveHandler(w http.ResponseWriter, r *http.Request, title string, config types.Config) {
	body := r.FormValue("body")
	p := &file.Page{Title: title, Raw: []byte(body)}
	err := file.Save(p, config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func DeleteHandler(w http.ResponseWriter, r *http.Request, title string, config types.Config) {
	ok, err := file.DeletePage(title, config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if ok {
		http.Redirect(w, r, "/view/Index", http.StatusFound)
	}
}

func EditHandler(w http.ResponseWriter, r *http.Request, title string, config types.Config) {
	p, err := file.LoadPage(title, config)
	if err != nil {
		p = &file.Page{Title: title}
	}
	file.RenderTemplate(w, "edit", p)
}

func FileUploadHandler(w http.ResponseWriter, r *http.Request, config types.Config) {
	r.ParseMultipartForm(10 << 20) // 10MB

	mpfile, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer mpfile.Close()

	// Now let’s save it locally
	dst, err := file.CreateFile(handler.Filename, config)
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the destination file
	if _, err := dst.ReadFrom(mpfile); err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
	}

	http.Redirect(w, r, "/special/Attachments", http.StatusFound)
}

func SpecialHandler(w http.ResponseWriter, r *http.Request, path string, config types.Config) {
	parts := strings.Split(path[1:], "/")
	command := parts[1]

	switch command {
	case "AllFiles":
		files := file.ListFiles(config)
		file.ShowFiles(w, types.TemplateView{Title: "All Files", Files: files, Special: true, SearchCriteria: ""})
	case "SearchFiles":
		criteria := r.FormValue("search")
		files, completeMatch := file.Search(file.ListFiles(config), criteria, config)
		caser := cases.Title(language.English)
		if !completeMatch {
			files = append(files, types.ExistingFile{FileName: caser.String(criteria), Exists: false})
		}
		title := fmt.Sprintf("Files found with '%s'", criteria)
		file.ShowFiles(w, types.TemplateView{Title: title, Files: files, Special: true, SearchCriteria: criteria})
	case "Attachments":
		file.ShowAttachments(w, config)
	case "DelAtt":
		dir := filepath.Join(config.DataDir, "att")
		filename := filepath.Join(dir, parts[2])
		os.Remove(filename)
		http.Redirect(w, r, "/special/Attachments", http.StatusFound)
	case "RandomFile":
		files := file.ListFiles(config)
		file := files[rand.IntN(len(files))]
		http.Redirect(w, r, "/view/"+file.FileName, http.StatusFound)
	default:
		http.Redirect(w, r, "/view/Index", http.StatusFound)
	}
}
