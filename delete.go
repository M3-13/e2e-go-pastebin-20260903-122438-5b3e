package main

import "net/http"

func DeletePasteHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !isValidHexID(id) {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if !store.Delete(id) {
		writeError(w, http.StatusNotFound, "paste not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// isValidHexID reports whether id is a non-empty string consisting solely of
// hexadecimal digits.
func isValidHexID(id string) bool {
	if id == "" {
		return false
	}
	for _, c := range id {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}
