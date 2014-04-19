package main

import (
  "encoding/csv"
  "encoding/json"
  "flag"
  "fmt"
  "io"
  "io/ioutil"
  "os"
  "path/filepath"
  "strconv"
)

type State struct {
  Code int
  Name string
  Abbr string
}

type Zip struct {
  Code string
  City string
  Pop  int
  Lat  float64
  Lon  float64
}

func LoadStateCodes(filename string) (map[int]*State, error) {
  r, err := os.Open(filename)
  if err != nil {
    return nil, err
  }
  defer r.Close()

  m := map[int]*State{}
  cr := csv.NewReader(r)
  for {
    rec, err := cr.Read()
    if err == io.EOF {
      return m, nil
    } else if err != nil {
      return nil, err
    }

    code, err := strconv.ParseInt(rec[2], 10, 64)
    if err != nil {
      return nil, err
    }

    m[int(code)] = &State{
      Code: int(code),
      Name: rec[0],
      Abbr: rec[1],
    }
  }
}

func LoadPopulations(filename string) (map[string]int, error) {
  r, err := os.Open(filename)
  if err != nil {
    return nil, err
  }
  defer r.Close()

  p := map[string]int{}
  cr := csv.NewReader(r)

  _, err = cr.Read()
  if err != nil {
    return nil, err
  }

  for {
    rec, err := cr.Read()
    if err == io.EOF {
      return p, nil
    } else if err != nil {
      return nil, err
    }

    pop, err := strconv.ParseInt(rec[1], 10, 64)
    if err != nil {
      return nil, err
    }

    p[rec[0]] = int(pop)
  }
}

func ParseLatLon(s string) (float64, error) {
  v, err := strconv.ParseFloat(s[1:], 64)
  if err != nil {
    return 0, err
  }
  if s[0] == '-' {
    v = -v
  }
  return v, nil
}

func LoadZips(filename string, states map[int]*State, pops map[string]int) ([]*Zip, error) {
  r, err := os.Open(filename)
  if err != nil {
    return nil, err
  }
  defer r.Close()

  var zips []*Zip
  cr := csv.NewReader(r)
  cr.Comma = '\t'
  for {
    rec, err := cr.Read()
    if err == io.EOF {
      return zips, nil
    } else if err != nil {
      return nil, err
    }

    lat, err := ParseLatLon(rec[1])
    if err != nil {
      return nil, err
    }

    lon, err := ParseLatLon(rec[2])
    if err != nil {
      return nil, err
    }

    sc, err := strconv.ParseInt(rec[5], 10, 64)
    if err != nil {
      return nil, err
    }

    zips = append(zips, &Zip{
      Code: rec[0],
      City: fmt.Sprintf("%s, %s", rec[4], states[int(sc)].Abbr),
      Pop:  pops[rec[0]],
      Lat:  lat,
      Lon:  lon,
    })
  }
}

func WriteJson(filename string, data interface{}) error {
  b, err := json.MarshalIndent(data, "", "  ")
  if err != nil {
    return err
  }

  return ioutil.WriteFile(filename, b, os.ModePerm)
}

func main() {
  flagWork := flag.String("work", "work", "")
  flagData := flag.String("data", "data", "")
  flag.Parse()

  states, err := LoadStateCodes(filepath.Join(*flagData, "states.csv"))
  if err != nil {
    panic(err)
  }

  pops, err := LoadPopulations(filepath.Join(*flagData, "population-by-zip.csv"))
  if err != nil {
    panic(err)
  }

  zips, err := LoadZips(filepath.Join(*flagData, "zips"), states, pops)
  if err != nil {
    panic(err)
  }

  os.MkdirAll(*flagWork, os.ModePerm)

  if err := WriteJson(filepath.Join(*flagWork, "zips.json"), zips); err != nil {
    panic(err)
  }
}
