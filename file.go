package main

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Page struct {
	Title   string
	Body    template.HTML
	Raw     []byte
	Special bool
}

type ExistingFile struct {
	FileName string
	Exists   bool
	Hits     int
}

func (p *Page) save(config Config) error {
	filename := filepath.Join(config.DataDir, p.Title+".md")
	return os.WriteFile(filename, p.Raw, 0600)
}

func deletePage(title string, config Config) (bool, error) {
	filename := filepath.Join(config.DataDir, title+".md")
	err := os.Remove(filename)

	if err != nil {
		return false, err
	}

	return true, nil
}

func loadPage(title string, config Config) (*Page, error) {
	filename := filepath.Join(config.DataDir, title+".md")
	raw, err := os.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	html := string(mdToHTML(raw))

	// subst
	found := links.FindAllString(html, -1)
	for _, link := range found {
		newlink := link[1 : len(link)-1]
		linkHtml := fmt.Sprintf("<a href=\"/view/%s\">%s</a>", newlink, newlink)
		html = strings.Replace(html, link, linkHtml, -1)
	}

	body := template.HTML(html)

	return &Page{Title: title, Body: body, Raw: raw, Special: false}, nil
}

func showFiles(w http.ResponseWriter, templateView TemplateView) {
	err := templates.ExecuteTemplate(w, "files.html", templateView)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func createFile(filename string, config Config) (*os.File, error) {
	dir := filepath.Join(config.DataDir, "att")
	file := filepath.Join(dir, filename)

	// Create an uploads directory if it doesn’t exist
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, 0755)
	}

	// Build the file path and create it
	dst, err := os.Create(file)
	if err != nil {
		return nil, err
	}

	return dst, nil
}

func listFiles(config Config) []ExistingFile {
	entries, err := os.ReadDir(config.DataDir)
	if err != nil {
		log.Fatal(err)
	}

	var files []ExistingFile
	for _, e := range entries {
		if !e.IsDir() {
			base := strings.Split(e.Name(), ".")[0]
			files = append(files, ExistingFile{FileName: base, Exists: true})
		}
	}

	return files
}

func listAttachments(config Config) []ExistingFile {
	dir := filepath.Join(config.DataDir, "att")
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	var files []ExistingFile
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, ExistingFile{FileName: e.Name(), Exists: true})
		}
	}

	return files
}

func search(list []ExistingFile, criteria string, config Config) ([]ExistingFile, bool) {
	var found []ExistingFile
	var completeMatch = false

	// filename matches
	for _, entry := range list {

		if len(criteria) > 2 {
			hits := contentSearch(entry, criteria, config)
			if hits > 0 {
				found = append(found, ExistingFile{FileName: entry.FileName, Exists: entry.Exists, Hits: hits})
			}
		}

		if strings.Contains(strings.ToLower(entry.FileName), strings.ToLower(criteria)) {
			if strings.EqualFold(strings.ToLower(entry.FileName), strings.ToLower(criteria)) {
				completeMatch = true
			}
			found = append(found, entry)
		}
	}

	return found, completeMatch
}

func contentSearch(file ExistingFile, criteria string, config Config) int {
	filename := filepath.Join(config.DataDir, file.FileName+".md")
	f, err := os.Open(filename)

	if err != nil {
		return 0
	}
	defer f.Close()

	// Splits on newlines by default.
	scanner := bufio.NewScanner(f)

	hits := 0

	for scanner.Scan() {
		if strings.Contains(strings.ToLower(scanner.Text()), strings.ToLower(criteria)) {
			hits++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0
	}

	return hits
}
