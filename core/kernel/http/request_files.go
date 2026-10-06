package http

// HasFileAny reports whether any of the given file keys are present.
func (r *Request) HasFileAny(keys ...string) bool {
	r.poisonCheck()
	for _, key := range keys {
		if r.HasFile(key) {
			return true
		}
	}
	return false
}
