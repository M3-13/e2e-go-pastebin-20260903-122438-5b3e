package main

import "net/http"

// store is the single in-memory store shared by all paste handlers.
var store = NewStore()

func ListPastesHandler(w http.ResponseWriter, r *http.Request) {
	metas := store.List()
	if metas == nil {
		metas = []PasteMeta{}
	}
	writeJSON(w, http.StatusOK, metas)
}
