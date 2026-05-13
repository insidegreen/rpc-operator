package api

import (
	"fmt"
	"net/http"
)

func (s *Server) handleCatalogList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.Catalog.All()})
}

func (s *Server) handleCatalogGet(w http.ResponseWriter, r *http.Request) {
	category := r.PathValue("category")
	name := r.PathValue("name")
	comp, ok := s.Catalog.Get(category, name)
	if !ok {
		writeJSONError(w, http.StatusNotFound,
			"component not found",
			fmt.Sprintf("%s/%s not in v0.2 starter catalog", category, name))
		return
	}
	writeJSON(w, http.StatusOK, comp)
}
