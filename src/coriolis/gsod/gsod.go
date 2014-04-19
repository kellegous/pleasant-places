package gsod

import (
  "archive/tar"
  "bufio"
  "bytes"
  "compress/gzip"
  "coriolis"
  "fmt"
  "io"
  "os"
  "path/filepath"
  "sort"
  "strconv"
  "strings"
  "time"
)

type Store struct {
  *coriolis.Store
  Years []int
}

type Summary struct {
  Station   *coriolis.Station
  Day       time.Time
  WindAvg   float64
  WindMax   float64
  TempAvg   float64
  TempMax   float64
  TempMin   float64
  Precip    float64
  SnowDepth float64
}

func toStationId(l string) string {
  return fmt.Sprintf("%s-%s", l[0:6], l[7:12])
}

func toSummary(s *Summary, station *coriolis.Station, l string) error {
  s.Station = station

  yr, err := strconv.ParseInt(l[14:18], 10, 32)
  if err != nil {
    return err
  }

  mn, err := strconv.ParseInt(l[18:20], 10, 32)
  if err != nil {
    return err
  }

  dy, err := strconv.ParseInt(l[20:22], 10, 32)
  if err != nil {
    return err
  }

  s.Day = time.Date(int(yr), time.Month(int(mn)), int(dy), 0, 0, 0, 0, time.UTC)

  var tmp float64
  tmp, err = strconv.ParseFloat(strings.TrimSpace(l[78:83]), 64)
  if err != nil {
    return err
  }
  s.WindAvg = tmp

  tmp, err = strconv.ParseFloat(strings.TrimSpace(l[88:93]), 64)
  if err != nil {
    return err
  }
  s.WindMax = tmp

  tmp, err = strconv.ParseFloat(strings.TrimSpace(l[24:30]), 64)
  if err != nil {
    return err
  }
  s.TempAvg = tmp

  tmp, err = strconv.ParseFloat(strings.TrimSpace(l[102:108]), 64)
  if err != nil {
    return err
  }
  s.TempMax = tmp

  tmp, err = strconv.ParseFloat(strings.TrimSpace(l[110:116]), 64)
  if err != nil {
    return err
  }
  s.TempMin = tmp

  tmp, err = strconv.ParseFloat(strings.TrimSpace(l[118:123]), 64)
  if err != nil {
    return err
  }
  s.Precip = tmp

  tmp, err = strconv.ParseFloat(strings.TrimSpace(l[125:130]), 64)
  if err != nil {
    return err
  }
  s.SnowDepth = tmp

  return nil
}

func (s *Store) ForEachSummaryInYear(year int, f func(s *Summary) error) error {
  var summary Summary
  return forEachSummary(
    filepath.Join(s.Dir, fmt.Sprintf("gsod_%d.tar", year)), s.StationIndex, &summary, f)
}

func forEachSummary(filename string, stations map[string]*coriolis.Station, s *Summary, fn func(s *Summary) error) error {
  r, err := os.Open(filename)
  if err != nil {
    return err
  }
  defer r.Close()

  tr := tar.NewReader(r)
  var buf bytes.Buffer
  for {
    // advance the tar to the next entry
    h, err := tr.Next()
    if err == io.EOF {
      return nil
    } else if err != nil {
      return err
    }

    // directory entries for . and .. are included, ignore those
    if h.FileInfo().IsDir() {
      continue
    }

    // each entry is compressed with gzip
    gr, err := gzip.NewReader(tr)
    if err != nil {
      return err
    }

    // read each line in the entry
    br := bufio.NewReader(gr)
    for {
      b, p, err := br.ReadLine()
      if err == io.EOF {
        break
      } else if err != nil {
        return err
      }

      buf.Write(b)
      if p {
        continue
      }

      // Is this a header?
      line := buf.String()
      buf.Reset()

      if strings.HasPrefix(line, "STN--- WBAN   YEARMODA") {
        continue
      }

      station := stations[toStationId(line)]
      if station == nil {
        continue
      }

      if err := toSummary(s, station, line); err != nil {
        return err
      }

      if err := fn(s); err != nil {
        return err
      }

    }
  }
}

func OpenStore(dir string) (*Store, error) {
  s, err := coriolis.OpenStore(dir)
  if err != nil {
    return nil, err
  }

  return NewStore(s)
}

func NewStore(s *coriolis.Store) (*Store, error) {
  files, err := filepath.Glob(filepath.Join(s.Dir, "gsod_*.tar"))
  if err != nil {
    return nil, err
  }

  years := make([]int, 0, len(files))
  for _, file := range files {
    base := filepath.Base(file)
    yr, err := strconv.ParseInt(base[5:9], 10, 64)
    if err != nil {
      return nil, err
    }
    years = append(years, int(yr))
  }

  sort.Ints(years)

  return &Store{
    Store: s,
    Years: years,
  }, nil
}
