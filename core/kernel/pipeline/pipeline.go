package pipeline

// Pipe transforms a passable value and calls next.
type Pipe func(passable any, next func(any) any) any

// Destination is the final callback.
type Destination func(passable any) any

// Pipeline sends a value through a series of pipes.
type Pipeline struct {
	passable any
	pipes    []Pipe
}

// Send starts a pipeline with the given passable.
func Send(passable any) *Pipeline {
	return &Pipeline{passable: passable, pipes: make([]Pipe, 0)}
}

// Through appends pipes.
func (p *Pipeline) Through(pipes ...Pipe) *Pipeline {
	p.pipes = append(p.pipes, pipes...)
	return p
}

// Pipe appends a single pipe.
func (p *Pipeline) Pipe(pipe Pipe) *Pipeline {
	p.pipes = append(p.pipes, pipe)
	return p
}

// Then runs the pipeline and returns the final result.
func (p *Pipeline) Then(destination Destination) any {
	pipeline := destination
	for i := len(p.pipes) - 1; i >= 0; i-- {
		pipe := p.pipes[i]
		next := pipeline
		pipeline = func(passable any) any {
			return pipe(passable, next)
		}
	}
	return pipeline(p.passable)
}

// ThenReturn runs pipes and returns the passable (identity destination).
func (p *Pipeline) ThenReturn() any {
	return p.Then(func(passable any) any { return passable })
}

// Via adapts a simple transformer into a Pipe.
func Via(fn func(any) any) Pipe {
	return func(passable any, next func(any) any) any {
		return next(fn(passable))
	}
}
