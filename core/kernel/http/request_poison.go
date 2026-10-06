package http

// poisonCheck panics on a poison build when the request was already released.
// requestFreedPoison is a const, so the production compiler deletes this body.
func (r *Request) poisonCheck() {
	if requestFreedPoison && r != nil && r.freed {
		panic("request used after handler returned")
	}
}

func (r *Request) markFreed() {
	if requestFreedPoison && r != nil {
		r.freed = true
	}
}
