#!/bin/bash

GOOS=linux GOARCH=arm64 go build 
tar -czf notesmd-Linux-arm64.tar.gz notesmd web notes

GOOS=linux GOARCH=amd64 go build
tar -czf notesmd-Linux-amd64.tar.gz notesmd web notes
