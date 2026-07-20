package deps

import (
	"io/fs"
	"net/http"
	"strings"
)

func spaHandler(assets fs.FS) http.Handler {
	fileServer := http.FileServerFS(assets)

	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		idx := r.Clone(r.Context())
		idx.URL.Path = "/"
		idx.URL.RawPath = ""

		fileServer.ServeHTTP(w, idx)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := strings.TrimPrefix(r.URL.Path, "/")
		if upath == "" {
			serveIndex(w, r)

			return
		}

		if f, err := assets.Open(upath); err == nil {
			_ = f.Close()

			fileServer.ServeHTTP(w, r)

			return
		}

		if strings.HasPrefix(upath, "assets/") {
			http.NotFound(w, r)

			return
		}

		serveIndex(w, r)
	})
}
