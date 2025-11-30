# NotesMD

Small webserver for wiki style editing of markdown notes.

Yes there are a lot of markdown style note taking applications and I tried a few and there is no ideal app for me so I figured, let's learn go and write one.

## Features

* Single user
* Markdown files (no database storage)
* Works like a wiki

## Building app

    go build notesmd.go

    If you use the buid.sh script it will build and package the app in a tar for Linux on amd64 & arm64.

## Running the application

    ./notesmd

### Arguments

    | Argument | Description |
    | ---- | ---- |
    | --data_dir | Directory to store the files in, 'notes' as default |
    | --port | Port to run the webserver on, 8080 as default |