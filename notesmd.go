package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
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

var validPath = regexp.MustCompile("^/(edit|save|view|special|delete)/([a-zA-Z0-9\\s]+)$")
var links = regexp.MustCompile("\\{([a-zA-Z0-9\\s]+)\\}")
var htmlFormatter *html.Formatter
var highlightStyle *chroma.Style

var tmplFiles = []string{
	"web/templates/header.html",
	"web/templates/menu.html",
	"web/templates/edit.html",
	"web/templates/view.html",
	"web/templates/allfiles.html",
}
var templates = template.Must(template.ParseFiles(tmplFiles...))

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

func htmlHighlight(w io.Writer, source, lang, defaultLang string) error {
	if lang == "" {
		lang = defaultLang
	}
	l := lexers.Get(lang)
	if l == nil {
		l = lexers.Analyse(source)
	}
	if l == nil {
		l = lexers.Fallback
	}
	l = chroma.Coalesce(l)

	it, err := l.Tokenise(nil, source)
	if err != nil {
		return err
	}

	return htmlFormatter.Format(w, highlightStyle, it)
}

func renderCode(w io.Writer, codeBlock *ast.CodeBlock, _ bool) {
	defaultLang := ""
	lang := string(codeBlock.Info)
	htmlHighlight(w, string(codeBlock.Literal), lang, defaultLang)
}

func myRenderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	if code, ok := node.(*ast.CodeBlock); ok {
		renderCode(w, code, entering)
		return ast.GoToNext, true
	}
	return ast.GoToNext, false
}

func mdToHTML(md []byte) []byte {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock | parser.DefinitionLists
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := mdhtml.CommonFlags | mdhtml.HrefTargetBlank | mdhtml.TOC
	opts := mdhtml.RendererOptions{
		Flags:          htmlFlags,
		RenderNodeHook: myRenderHook,
	}
	renderer := mdhtml.NewRenderer(opts)

	return markdown.Render(doc, renderer)
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

func specialHandler(w http.ResponseWriter, r *http.Request, title string, config Config) {
	switch title {
	case "AllFiles":
		files := listFiles(config)
		err := templates.ExecuteTemplate(w, "allfiles.html", struct {
			Title string
			Files []string
		}{Title: "", Files: files})
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
	dataDir := flag.String("data_dir", "notes", "Path to the directory where all the markdown files are stored.")
	port := flag.Int("port", 8080, "Port to run the webserver on.")
	style := flag.String("code_style", "monokai", "Code highlighting format to use; default is Monokai")

	flag.Parse()

	fmt.Printf("NotesMD, running on port %d, using directory %s, using code style %s\n", *port, *dataDir, *style)

	htmlFormatter = html.New(html.TabWidth(2), html.PreventSurroundingPre(false))

	if htmlFormatter == nil {
		panic("couldn't create html formatter")
	}
	highlightStyle = styles.Get(*style)

	if highlightStyle == nil {
		panic(fmt.Sprintf("didn't find style '%s'", highlightStyle.Name))
	}

	config := Config{DataDir: "undefined"}
	config.DataDir = *dataDir

	http.HandleFunc("/view/", makeHandler(viewHandler, config))
	http.HandleFunc("/edit/", makeHandler(editHandler, config))
	http.HandleFunc("/save/", makeHandler(saveHandler, config))
	http.HandleFunc("/delete/", makeHandler(deleteHandler, config))
	http.HandleFunc("/special/", makeHandler(specialHandler, config))

	http.Handle("/web/", http.StripPrefix("/web", http.FileServer(http.Dir("./web"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/view/Index", http.StatusFound)
	})

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}
