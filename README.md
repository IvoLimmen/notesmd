# NotesMD

Small webserver for wiki style editing of markdown notes.

Yes there are a lot of markdown style note taking applications and I tried a few and there is no ideal app for me so I figured, let's learn go and write one.

## Features

* Single user
* Markdown files (no database storage)
* Works like a wiki, if a page does not exist it opens the editor
* Has syntax highlighting for code blocks.
* Simple file search
* Editing page is simple but has a nice cheatsheet

## Building app

    go build notesmd.go

    If you use the buid.sh script it will build and package the app in a tar for Linux on amd64 & arm64.

## Running the application

    ./notesmd

### Arguments

    --data_dir, Directory to store the files in, 'notes' as default
    --port, Port to run the webserver on, 8080 as default
    --code_style, syntax highlighting style with 'Monokai' as default

## Credits

* It's completely based on the excellent tutorial: https://go.dev/doc/articles/wiki/.
* Uses github.com/alecthomas/chroma/v2 for syntax highlighting.
* Uses github.com/gomarkdown/markdown for the markdown parsing and formatting.
