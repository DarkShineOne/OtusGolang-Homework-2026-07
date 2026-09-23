package hw06pipelineexecution

type (
	In  = <-chan interface{}
	Out = In
	Bi  = chan interface{}
)

type Stage func(in In) (out Out)

func forward(src In, done In) Out {
	out := make(Bi)

	go func() {
		canceling := false
		for {
			if canceling {
				if _, ok := <-src; !ok {
					return
				}
				continue
			}
			select {
			case <-done:
				close(out)
				canceling = true
			case v, ok := <-src:
				if !ok {
					close(out)
					return
				}
				select {
				case out <- v:
				case <-done:
					close(out)
					canceling = true
				}
			}
		}
	}()

	return out
}

func ExecutePipeline(in In, done In, stages ...Stage) Out {
	cur := in

	for _, stage := range stages {
		cur = stage(forward(cur, done))
	}

	return forward(cur, done)
}
