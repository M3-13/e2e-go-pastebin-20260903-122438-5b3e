package main

import "net/http"

func DeletePasteHandler(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
