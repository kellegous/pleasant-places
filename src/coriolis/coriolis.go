package coriolis

import (
  "encoding/csv"
  "fmt"
  "io"
  "os"
  "path/filepath"
  "strconv"
  "strings"
)

const HistoryFile = "ish-history.csv"

type Station struct {
  Usaf     string
  Wban     string
  Name     string
  Call     string
  Country  string
  State    string
  Lat, Lon float64
}

func (s *Station) Id() string {
  return fmt.Sprintf("%s-%s", s.Usaf, s.Wban)
}

type Store struct {
  Dir          string
  Stations     []*Station
  StationIndex map[string]*Station
}

func OpenStore(dir string) (*Store, error) {
  // load stations
  stations, err := LoadStations(dir, InContinentalUs)
  if err != nil {
    return nil, err
  }

  // build index
  index := map[string]*Station{}
  for _, station := range stations {
    index[station.Id()] = station
  }

  return &Store{
    Dir:          dir,
    Stations:     stations,
    StationIndex: index,
  }, nil
}

func parseLatLon(s string, v *float64) error {
  *v = 0
  s = strings.TrimSpace(s)
  if s == "" {
    return nil
  }

  n, err := strconv.ParseInt(s[1:], 10, 64)
  if err != nil {
    return err
  }

  if s[0] == '-' {
    n = -n
  }

  *v = float64(n) / 1000
  return nil
}

func ForEachStation(dir string, fn func(s *Station) error) error {
  r, err := os.Open(filepath.Join(dir, HistoryFile))
  if err != nil {
    return err
  }
  defer r.Close()

  cr := csv.NewReader(r)

  // ditch the headers
  if _, err := cr.Read(); err != nil {
    return err
  }

  var s Station
  for {
    v, err := cr.Read()
    if err == io.EOF {
      return nil
    } else if err != nil {
      return err
    }

    s.Usaf = v[0]
    s.Wban = v[1]
    s.Name = v[2]
    s.Country = v[3]
    s.State = v[5]
    s.Call = v[6]

    if err := parseLatLon(v[7], &s.Lat); err != nil {
      return err
    }

    if err := parseLatLon(v[8], &s.Lon); err != nil {
      return err
    }

    if err := fn(&s); err != nil {
      return err
    }
  }
}

func stationInUsBounds(s *Station) bool {
  if s.Lat == 0 || s.Lon == 0 {
    return false
  }

  if int(s.Lat) == -99 || int(s.Lon) == -99 {
    return false
  }

  if s.Lat < 20 {
    return false
  }

  if s.Lon > -60 || s.Lon < -130 {
    return false
  }

  return true
}

func InContinentalUs(s *Station) bool {
  // we are only interested in stations in the continental US
  // ... that are not offshore buoys
  // ... and sit in the right geo bounds
  // ... and is not dinner key afb
  // ... and is not the bodega bay light house
  if s.Country != "US" || s.State == "AK" || s.State == "HI" ||
    strings.Contains(s.Name, "BUOY") ||
    !stationInUsBounds(s) ||
    s.Wban == "12848" || s.Usaf == "724995" {
    return false
  }
  return true
}

func LoadStations(filename string, fn func(s *Station) bool) ([]*Station, error) {
  var stations []*Station
  if err := ForEachStation(filename, func(s *Station) error {
    if !fn(s) {
      return nil
    }

    var ns Station = *s
    stations = append(stations, &ns)
    return nil
  }); err != nil {
    return nil, err
  }

  return stations, nil
}
