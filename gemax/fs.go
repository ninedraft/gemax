package gemax

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/ninedraft/gemax/gemax/status"
)

// FileSystem serves file systems as gemini catalogs.
// It will search index.gmi and index.gemini in each catalog
// to use it content as header of corresponding directory page.
//
// This handler is not intended to be used as file server,
// it's more like a static site server.
type FileSystem struct {
	// The backend file system.
	FS fs.FS

	// Will be prepended to the request pats.
	Prefix string
	// Optional text logger.
	Logf func(format string, args ...interface{})
}

var _ Handler = new(FileSystem).Serve

// Serve provided file system as gemini catalogs.
func (fileSystem *FileSystem) Serve(ctx context.Context, rw ResponseWriter, req IncomingRequest) {
	fileSystem.logf("INFO: %s is requested from %s", req.URL(), req.RemoteAddr())

	var p = path.Join(fileSystem.Prefix, req.URL().Path)
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		p = "."
	}
	fileSystem.logf("INFO: serving path: %s", p)

	var file, errOpen = fileSystem.FS.Open(p)
	switch {
	case errors.Is(errOpen, fs.ErrNotExist):
		const code = status.NotFound
		fileSystem.logf("WARN: %s is not found", p)
		rw.WriteStatus(code, code.String()+": "+req.URL().Path)
		return
	case errOpen != nil:
		fileSystem.logf("ERROR: serving %s: opening file: %v", p, errOpen)
		rw.WriteStatus(status.ServerUnavailable, "")
		return
	}
	defer func() { _ = file.Close() }()

	var info, errInfo = file.Stat()
	if errInfo != nil {
		rw.WriteStatus(status.ServerUnavailable, "")
		return
	}
	switch {
	case info.IsDir() && !isDirReader(file):
		fileSystem.logf("ERROR: serving dir %s: file is not ad directory reader!", p)
		rw.WriteStatus(status.ServerUnavailable, "")
	case info.IsDir() && isDirReader(file):
		_ = file.Close()
		fileSystem.serveDir(rw, req, p)
	default:
		fileSystem.serveFile(rw, p, file)
	}
}

func (fileSystem *FileSystem) serveFile(rw ResponseWriter, name string, file io.Reader) {
	var errHead = fileSystem.serveFileHead(rw, name, file)
	if errHead != nil {
		fileSystem.logf("serving file %s: reading file head: %v", name, errHead)
		rw.WriteStatus(status.ServerUnavailable, "")
		return
	}
	var _, errCopy = io.Copy(rw, file)
	if errCopy != nil {
		fileSystem.logf("ERROR: serving file %s: %v", name, errCopy)
	}
	fileSystem.logf("INFO: serving file %s: ok", name)
}

func (fileSystem *FileSystem) serveFileHead(rw ResponseWriter, name string, file io.Reader) error {
	var ext = path.Ext(name)
	if ext == ".gmi" || ext == ".gemini" {
		rw.WriteStatus(status.Success, MIMEGemtext)
		return nil
	}
	var buf = make([]byte, 512)
	var n, errHead = file.Read(buf)
	if errHead != nil {
		rw.WriteStatus(status.ServerUnavailable, "")
		return errHead
	}
	buf = buf[:n]
	var contentType = http.DetectContentType(buf)
	rw.WriteStatus(status.Success, contentType)
	var _, errWrite = rw.Write(buf)
	return errWrite
}

func (fileSystem *FileSystem) serveDir(rw ResponseWriter, req IncomingRequest, dir string) {
	fileSystem.serveIndexFile(rw, dir, "index.gmi", "index.gemini")
	var entries, errEntries = fs.ReadDir(fileSystem.FS, dir)
	if errEntries != nil {
		fileSystem.Logf("ERROR: serving dir %s: reading dir content: %v", dir, errEntries)
		rw.WriteStatus(status.ServerUnavailable, "")
		return
	}
	_, _ = rw.Write([]byte("\r\n"))
	for _, entry := range entries {
		var fileLink = path.Join(req.URL().Path, entry.Name())
		var _, errWriteEntry = fmt.Fprintf(rw, "=> %s %s \r\n", fileLink, entry.Name())
		if errWriteEntry != nil {
			fileSystem.logf("ERROR: serving dir %s: writing file entry %s: %v", dir, entry.Name(), errWriteEntry)
			return
		}
	}
}

func (fileSystem *FileSystem) serveIndexFile(rw ResponseWriter, dir string, names ...string) {
	for _, name := range names {
		var indexFile = path.Join(dir, name)
		var data, err = fs.ReadFile(fileSystem.FS, indexFile)
		if err != nil {
			fileSystem.logf("WARN: serving dir %s: searching index file %s: %v", dir, indexFile, err)
			continue
		}
		fileSystem.serveFile(rw, indexFile, bytes.NewReader(data))
		return
	}
	fileSystem.logf("WARN: serving dir %s: searching index files %v: no index files were found", dir, names)
}

func (fileSystem *FileSystem) logf(format string, args ...interface{}) {
	if fileSystem.Logf != nil {
		fileSystem.Logf(format, args...)
	}
}

func isDirReader(file fs.File) bool {
	var _, ok = file.(fs.ReadDirFile)
	return ok
}
