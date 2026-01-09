package file

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ivolimmen/notesmd/types"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

var htmlFormatter *html.Formatter
var highlightStyle *chroma.Style

var links = regexp.MustCompile(`{([a-zA-Z0-9\s]+)}`)

var tmplFiles = []string{
	"web/templates/header.html",
	"web/templates/menu.html",
	"web/templates/edit.html",
	"web/templates/view.html",
	"web/templates/files.html",
	"web/templates/attachments.html",
}
var templates = template.Must(template.ParseFiles(tmplFiles...))

type Page struct {
	Title   string
	Body    template.HTML
	Raw     []byte
	Special bool
}

func InitConfig(dataDir string, style string) types.Config {
	htmlFormatter = html.New(html.TabWidth(2), html.PreventSurroundingPre(false))

	if htmlFormatter == nil {
		panic("couldn't create html formatter")
	}
	highlightStyle = styles.Get(style)

	if highlightStyle == nil {
		panic(fmt.Sprintf("didn't find style '%s'", highlightStyle.Name))
	}

	config := types.Config{DataDir: "undefined"}
	config.DataDir = dataDir

	return config
}

func Save(p *Page, config types.Config) error {
	filename := filepath.Join(config.DataDir, p.Title+".md")
	return os.WriteFile(filename, p.Raw, 0600)
}

func DeletePage(title string, config types.Config) (bool, error) {
	filename := filepath.Join(config.DataDir, title+".md")
	err := os.Remove(filename)

	if err != nil {
		return false, err
	}

	return true, nil
}

func RenderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func LoadPage(title string, config types.Config) (*Page, error) {
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

func ShowFiles(w http.ResponseWriter, templateView types.TemplateView) {
	err := templates.ExecuteTemplate(w, "files.html", templateView)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func ShowAttachments(w http.ResponseWriter, config types.Config) {
	files := ListAttachments(config)
	err := templates.ExecuteTemplate(w, "attachments.html", types.TemplateView{Title: "Attachments", Files: files, Special: true, SearchCriteria: ""})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func CreateFile(filename string, config types.Config) (*os.File, error) {
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

func ListFiles(config types.Config) []types.ExistingFile {
	entries, err := os.ReadDir(config.DataDir)
	if err != nil {
		log.Fatal(err)
	}

	var files []types.ExistingFile
	for _, e := range entries {
		if !e.IsDir() {
			base := strings.Split(e.Name(), ".")[0]
			files = append(files, types.ExistingFile{FileName: base, Exists: true})
		}
	}

	return files
}

func ListAttachments(config types.Config) []types.ExistingFile {
	dir := filepath.Join(config.DataDir, "att")
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	var files []types.ExistingFile
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, types.ExistingFile{FileName: e.Name(), Exists: true})
		}
	}

	return files
}

func Search(list []types.ExistingFile, criteria string, config types.Config) ([]types.ExistingFile, bool) {
	var found []types.ExistingFile
	var completeMatch = false

	// filename matches
	for _, entry := range list {

		if strings.Contains(strings.ToLower(entry.FileName), strings.ToLower(criteria)) {
			if strings.EqualFold(strings.ToLower(entry.FileName), strings.ToLower(criteria)) {
				completeMatch = true
			}
			found = append(found, entry)
		} else {
			if len(criteria) > 2 {
				hits := contentSearch(entry, criteria, config)
				if hits > 0 {
					found = append(found, types.ExistingFile{FileName: entry.FileName, Exists: entry.Exists, Hits: hits})
				}
			}
		}
	}

	sort.Sort(types.ByName(found))

	return found, completeMatch
}

func contentSearch(file types.ExistingFile, criteria string, config types.Config) int {
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
