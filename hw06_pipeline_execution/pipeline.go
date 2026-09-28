package hw06pipelineexecution

type (
	In  = <-chan interface{}
	Out = In
	Bi  = chan interface{}
)

type Stage func(in In) (out Out)

func forward(src In, done In, drainOnDone bool) Out {
	out := make(Bi)

	go func() {
		for {
			select {
			case <-done:
				close(out)
				if drainOnDone {
					for range src { //nolint:revive
					}
				}
				return
			case v, ok := <-src:
				if !ok {
					close(out)
					return
				}
				select {
				case out <- v:
				case <-done:
					close(out)
					return
				}
			}
		}
	}()

	return out
}

func ExecutePipeline(in In, done In, stages ...Stage) Out {
	cur := forward(in, done, false)

	for _, stage := range stages {
		cur = stage(forward(cur, done, true))
	}

	return forward(cur, done, true)
}
