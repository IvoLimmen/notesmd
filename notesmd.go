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
	"web/templates/files.html",
}
var templates = template.Must(template.ParseFiles(tmplFiles...))

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

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func specialHandler(w http.ResponseWriter, r *http.Request, title string, config Config) {
	switch title {
	case "AllFiles":
		files := listFiles(config)
		showFiles(w, files, "All files")
	case "SearchFiles":
		criteria := r.FormValue("search")
		files := search(listFiles(config), criteria)
		title := fmt.Sprintf("Files found with '%s'", criteria)
		showFiles(w, files, title)
	case "RandomFile":
		files := listFiles(config)
		file := files[rand.IntN(len(files))]
		http.Redirect(w, r, "/view/"+file, http.StatusFound)
	default:
		http.Redirect(w, r, "/view/Index", http.StatusFound)
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

func search(list []string, criteria string) []string {
	var found []string
	for _, entry := range list {
		if strings.Contains(strings.ToLower(entry), strings.ToLower(criteria)) {
			found = append(found, entry)
		}
	}
	return found
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
