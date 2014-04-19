package main

import (
  "flag"
  "github.com/kellegous/pork"
  "log"
  "net/http"
  "os"
  "path/filepath"
  "runtime"
)

func rootPath() (string, error) {
  // try to resolve root from argv[0]
  d, err := filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), ".."))
  if err == nil {
    if _, err := os.Stat(filepath.Join(d, ".gaan")); err == nil {
      return d, nil
    }
  }

  // next try to resolve root from src
  _, file, _, _ := runtime.Caller(0)
  return filepath.Abs(filepath.Join(filepath.Dir(file), "../.."))
}

func main() {
  flagWork := flag.String("work", "work", "")
  flagAddr := flag.String("addr", ":4020", "")
  flag.Parse()

  root, err := rootPath()
  if err != nil {
    panic(err)
  }

  r := pork.NewRouter(
    func(status int, r *http.Request) {
      log.Printf("%d - %s", status, r.URL.Path)
    }, nil, nil)

  r.RespondWith("/", pork.Content(pork.NewConfig(pork.None),
    http.Dir(filepath.Join(root, "src/pub"))))
  r.RespondWith("/data/", pork.Content(pork.NewConfig(pork.None),
    http.Dir(*flagWork)))

  log.Printf("Server running on address %s\n", *flagAddr)
  if err := http.ListenAndServe(*flagAddr, r); err != nil {
    panic(err)
  }
}
