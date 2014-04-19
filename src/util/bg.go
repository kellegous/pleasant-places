package util

type Worker struct {
  n int
  c chan func() error
  e chan bool
}

func StartWorker(n int) *Worker {
  c := make(chan func() error)
  e := make(chan bool)

  for i := 0; i < n; i++ {
    go func() {
      for f := range c {
        if err := f(); err != nil {
          panic(err)
        }
      }

      e <- true
    }()
  }

  return &Worker{
    n: n,
    c: c,
    e: e,
  }
}

func (w *Worker) Do(f func() error) {
  w.c <- f
}

func (w *Worker) WaitForExit() {
  close(w.c)
  for i := 0; i < w.n; i++ {
    <-w.e
  }
}
