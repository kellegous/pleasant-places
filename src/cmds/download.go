package main

import (
  "flag"
  "fmt"
  "os"
  "os/exec"
  "path/filepath"
  "sort"
  "strconv"
  "strings"
  "time"
)

func EnsureDownload(dir, uri string) error {
  return EnsureDownloadTo(filepath.Join(dir, filepath.Base(uri)), uri)
}

func EnsureDownloadTo(dst, uri string) error {
  if _, err := os.Stat(dst); err == nil {
    // update the modification time on the files.
    t := time.Now()
    return os.Chtimes(dst, t, t)
  }

  os.MkdirAll(filepath.Dir(dst), os.ModePerm)

  fmt.Printf("%s\n", uri)

  // use curl to easily handle ftp:// urls
  return exec.Command("curl", "-o", dst, uri).Run()
}

func EnsureDownloadAll(dir string, uris ...string) error {
  for _, uri := range uris {
    if err := EnsureDownload(dir, uri); err != nil {
      return err
    }
  }
  return nil
}

func DownloadGsod(dst string, yr int) error {
  return EnsureDownloadTo(
    filepath.Join(dst, fmt.Sprintf("gsod_%d.tar", yr)),
    fmt.Sprintf("ftp://ftp.ncdc.noaa.gov/pub/data/gsod/%d/gsod_%d.tar", yr, yr))
}

func YearsFromArgs(args []string) ([]int, error) {
  ym := map[int]bool{}

  for _, arg := range args {
    v := strings.Split(arg, "-")
    switch len(v) {
    case 1:
      yr, err := strconv.ParseInt(arg, 10, 64)
      if err != nil {
        return nil, err
      }
      ym[int(yr)] = true
    case 2:
      b, err := strconv.ParseInt(v[0], 10, 64)
      if err != nil {
        return nil, err
      }

      e, err := strconv.ParseInt(v[1], 10, 64)
      if err != nil {
        return nil, err
      }

      for i := int(b); i <= int(e); i++ {
        ym[i] = true
      }
    default:
      return nil, fmt.Errorf("invalid argument: %s", arg)
    }
  }

  years := make([]int, 0, len(ym))
  for k, _ := range ym {
    years = append(years, k)
  }

  sort.Ints(years)

  return years, nil
}

func main() {
  flagDest := flag.String("dest", "data", "directory into which to download")
  flag.Parse()

  os.MkdirAll(*flagDest, os.ModePerm)

  if err := EnsureDownloadAll(*flagDest,
    "ftp://ftp.ncdc.noaa.gov/pub/data/noaa/isd-history.csv",
    "ftp://ftp.ncdc.noaa.gov/pub/data/noaa/isd-inventory.csv.z"); err != nil {
    panic(err)
  }

  years, err := YearsFromArgs(flag.Args())
  if err != nil {
    panic(err)
  }

  for _, year := range years {
    if err := DownloadGsod(*flagDest, year); err != nil {
      panic(err)
    }
  }
}
