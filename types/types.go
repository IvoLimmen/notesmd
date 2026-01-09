package types

type TemplateView struct {
	Title          string
	Files          []ExistingFile
	Special        bool
	SearchCriteria string
}

type Config struct {
	DataDir  string
	AllFiles []string
}

type ExistingFile struct {
	FileName string
	Exists   bool
	Hits     int
}

type ByName []ExistingFile

func (a ByName) Len() int           { return len(a) }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool { return a[i].FileName < a[j].FileName }
