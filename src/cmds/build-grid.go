package main

import (
  "coriolis"
  "coriolis/gsod"
  "encoding/json"
  "flag"
  "fmt"
  "image"
  "math"
  "os"
  "path/filepath"
  "util"
)

// A temperature preference range.
type TempPref struct {
  AvgMin float64
  AvgMax float64
  AbsMin float64
  AbsMax float64
  Name   string
}

var (
  // the normal temperature range
  LikeItNorm = &TempPref{55, 75, 45, 85, "norm"}

  // for those that like it a little warmer
  LikeItWarm = &TempPref{65, 85, 55, 95, "warm"}

  // for those that like it a little cooler
  LikeItCool = &TempPref{45, 65, 35, 65, "cool"}
)

// All known temperature preferences
var TempPrefs = []*TempPref{
  // LikeItCool,
  LikeItNorm,
  // LikeItWarm,
}

// Declares a grid of certain width & height with a list of
// active cells. This is used to selectively turn off parts
// of the grid.
type GridConfig struct {
  W      int
  H      int
  Size   int
  Active [][]int
}

// The location of a station in virtual coordinates
type StationLoc struct {
  *coriolis.Station
  X float64
  Y float64
}

// Defines a region of the grid and keeps a lot of
// data about what is contained within and around
// that grid.
type Region struct {
  I        int
  J        int
  Rect     image.Rectangle
  Stations []*StationLoc
  Nearest  []*StationLoc
  Zips     []*Zip
  City     string
}

// Essentially a matrix of regions. Inactive regions
// are nil.
type Grid struct {
  W    int
  H    int
  Grid [][]*Region
}

// Simple euclidean distance.
func Distance(x0, y0, x1, y1 float64) float64 {
  dx, dy := x1-x0, y1-y0
  return math.Sqrt(dx*dx + dy*dy)
}

// Sort a list of station locations by their distance to a particular point.
func SortStationLocs(locs []*StationLoc, x, y float64) {
  dist := make([]float64, len(locs))
  for i, loc := range locs {
    dist[i] = Distance(loc.X, loc.Y, x, y)
  }

  util.Sort(len(locs),
    func(i, j int) bool {
      return dist[i] < dist[j]
    },
    func(i, j int) {
      locs[i], locs[j] = locs[j], locs[i]
      dist[i], dist[j] = dist[j], dist[i]
    })
}

// Find the stations nearest the region (i,j) searching in radius r.
func (g *Grid) StationsAround(locs *[]*StationLoc, i, j, r int) {
  if i < 0 || j < 0 {
    return
  }

  if i >= g.W || j >= g.H {
    return
  }

  reg := g.Grid[i][j]
  if r == 0 {
    if reg != nil {
      for _, loc := range reg.Stations {
        *locs = append(*locs, loc)
      }
    }
    return
  }

  // top & bottom row
  for k := -r; k <= r; k++ {
    g.StationsAround(locs, i+k, j-r, 0)
    g.StationsAround(locs, i+k, j+r, 0)
  }

  n := r - 1
  // left & right cols
  for k := -n; k <= n; k++ {
    g.StationsAround(locs, i-r, j+k, 0)
    g.StationsAround(locs, i+r, j+k, 0)
  }
}

// Locate the major (most populous) city with region (i,j). This returns
// the name of the city and the population. If (i,j) is an inactive region,
// this will simply return "" and 0.
func MajorCityOf(g *Grid, i, j int) (string, int) {
  r := g.Grid[i][j]
  if r == nil {
    return "", 0
  }

  // some regions have zips with the same city name, combine them.
  m := map[string]int{}
  for _, zip := range r.Zips {
    m[zip.City] += zip.Pop
  }

  var name string
  var pop int

  for k, v := range m {
    if v > pop {
      name = k
      pop = v
    }
  }

  return name, pop
}

// Produce a text label for the region (i,j). First try to find a major
// city in the region. If there is not one, we're going to try to produce
// a relative name (i.e. EAST OF RENO).
func LabelFor(g *Grid, i, j int) string {
  name, _ := MajorCityOf(g, i, j)
  if name != "" {
    return name
  }

  var names []string
  var pops []int

  lookWest, lookEast, lookNorth, lookSouth := i > 0, i < (g.W-1), j > 0, j < (g.H-1)

  // east?
  if lookWest {
    if name, pop := MajorCityOf(g, i-1, j); name != "" {
      names = append(names, fmt.Sprintf("EAST OF %s", name))
      pops = append(pops, pop)
    }

    if lookNorth {
      if name, pop := MajorCityOf(g, i-1, j-1); name != "" {
        names = append(names, fmt.Sprintf("SE OF %s", name))
        pops = append(pops, pop)
      }
    }

    if lookSouth {
      if name, pop := MajorCityOf(g, i-1, j+1); name != "" {
        names = append(names, fmt.Sprintf("NE OF %s", name))
        pops = append(pops, pop)
      }
    }
  }

  // west?
  if lookEast {
    if name, pop := MajorCityOf(g, i+1, j); name != "" {
      names = append(names, fmt.Sprintf("WEST OF %s", name))
      pops = append(pops, pop)
    }

    if lookNorth {
      if name, pop := MajorCityOf(g, i+1, j-1); name != "" {
        names = append(names, fmt.Sprintf("SW OF %s", name))
        pops = append(pops, pop)
      }
    }

    if lookSouth {
      if name, pop := MajorCityOf(g, i+1, j+1); name != "" {
        names = append(names, fmt.Sprintf("NW OF %s", name))
        pops = append(pops, pop)
      }
    }
  }

  // south?
  if lookNorth {
    if name, pop := MajorCityOf(g, i, j-1); name != "" {
      names = append(names, fmt.Sprintf("SOUTH OF %s", name))
      pops = append(pops, pop)
    }
  }

  // north?
  if lookSouth {
    if name, pop := MajorCityOf(g, i, j+1); name != "" {
      names = append(names, fmt.Sprintf("NORTH OF %s", name))
      pops = append(pops, pop)
    }
  }

  if len(names) == 0 {
    return ""
  }

  util.Sort(len(names),
    func(i, j int) bool {
      return pops[j] < pops[i]
    }, func(i, j int) {
      names[i], names[j] = names[j], names[i]
      pops[i], pops[j] = pops[j], pops[i]
    })

  return names[0]
}

// Find the nearest N stations to the region r. This will search in an increasing
// radius in and around a region until N stations are found.
func NearestN(c *GridConfig, grid *Grid, r *Region, n int) []*StationLoc {
  cx := float64(r.Rect.Min.X) + float64(r.Rect.Dx())/2.0
  cy := float64(r.Rect.Min.Y) + float64(r.Rect.Dy())/2.0

  var locs []*StationLoc
  rad := 0
  for len(locs) < n {
    grid.StationsAround(&locs, r.I, r.J, rad)
    rad++
  }

  SortStationLocs(locs, cx, cy)

  return locs[0:n]
}

// Load the grid config from a file.
func LoadGridConfig(filename string, c *GridConfig) error {
  r, err := os.Open(filename)
  if err != nil {
    return err
  }
  defer r.Close()

  if err := json.NewDecoder(r).Decode(c); err != nil {
    return err
  }

  return nil
}

// Compute a transformation function that will take a point in lat, lon and produce a virtual x, y that fits relatively
// in the Rectangle r. TODO(knorton): This is a stupid and lazy projection, but it works.
func ComputeTransform(stations []*coriolis.Station, r image.Rectangle) func(float64, float64) (float64, float64) {
  minY, maxY, minX, maxX := stations[0].Lat, stations[0].Lat, stations[0].Lon, stations[0].Lon
  for _, s := range stations {
    if s.Lat > maxY {
      maxY = s.Lat
    } else if s.Lat < minY {
      minY = s.Lat
    } else if s.Lon > maxX {
      maxX = s.Lon
    } else if s.Lon < minX {
      minX = s.Lon
    }
  }

  f := float64(r.Dx()) / (maxX - minX)
  return func(lat, lon float64) (float64, float64) {
    return (lon - minX) * f, (maxY - lat) * f
  }
}

// Use the transformation to place each of the stations into a location.
func PlaceStations(stations []*coriolis.Station, tx func(float64, float64) (float64, float64)) []*StationLoc {
  var locs []*StationLoc
  for _, s := range stations {
    x, y := tx(s.Lat, s.Lon)
    locs = append(locs, &StationLoc{
      Station: s,
      X:       x,
      Y:       y,
    })
  }
  return locs
}

// Take the list of station locations and fold them into the regions of a grid.
func BuildGrid(locs []*StationLoc, zips []*Zip, r image.Rectangle, c *GridConfig) *Grid {
  nx, ny := c.W, c.H
  regs := make([][]*Region, nx)
  for i := 0; i < nx; i++ {
    regs[i] = make([]*Region, ny)
  }

  // create all active rectangles
  for _, xy := range c.Active {
    i, j := xy[0], xy[1]
    regs[i][j] = &Region{
      I:    i,
      J:    j,
      Rect: image.Rect(i*c.Size, j*c.Size, (i+1)*c.Size, (j+1)*c.Size),
    }
  }

  // assign stations to their region
  for _, loc := range locs {
    ix, iy := int(loc.X/float64(c.Size)), int(loc.Y/float64(c.Size))
    r := regs[ix][iy]
    if r == nil {
      continue
    }
    r.Stations = append(r.Stations, loc)
  }

  // assign zips to their region
  for _, zip := range zips {
    ix, iy := int(zip.X/float64(c.Size)), int(zip.Y/float64(c.Size))
    if ix >= c.W || ix < 0 || iy >= c.H || iy < 0 {
      continue
    }
    r := regs[ix][iy]
    if r == nil {
      continue
    }
    r.Zips = append(r.Zips, zip)
  }

  grid := &Grid{
    W:    nx,
    H:    ny,
    Grid: regs,
  }

  for _, xy := range c.Active {
    i, j := xy[0], xy[1]
    regs[i][j].Nearest = NearestN(c, grid, regs[i][j], 20)
    city := LabelFor(grid, i, j)
    regs[i][j].City = city
  }

  return grid
}

// Utility method for serializing an object to JSON and writing
// it to a file.
func WriteJson(filename string, data interface{}) error {
  w, err := os.Create(filename)
  if err != nil {
    return err
  }
  defer w.Close()

  return json.NewEncoder(w).Encode(data)
}

// Determine if the summary data indicates a "pleasant" day according to the
// given temperature prefs.
func IsPleasant(s *gsod.Summary, p *TempPref) bool {
  if s.TempAvg < 999 && (s.TempAvg < p.AvgMin || s.TempAvg > p.AvgMax) {
    return false
  }

  if s.TempMax < 999 && s.TempMax > p.AbsMax {
    return false
  }

  if s.TempMin < p.AbsMin {
    return false
  }

  if s.Precip < 99 && s.Precip > 0.000001 {
    return false
  }

  if s.SnowDepth < 999 && s.SnowDepth > 0.25 {
    return false
  }

  return true
}

// A rational type holding a perctage value, that can also
// represent NaN as zero.
type Pct struct {
  A, B int
}

// Turn the number into a byte (0-255), 0 is the null value.
func (p *Pct) Byte() byte {
  if p.B == 0 {
    return 0
  }
  f := 255.0 * float64(p.A) / float64(p.B)
  if f >= 255 {
    return 255
  }
  return 1 + byte(f)
}

// The summarized stats for a region.
type RegionStats struct {
  *Region
  Months [12]byte
  Total  byte
}

// A customized JSON marshaler.
func (r *RegionStats) MarshalJSON() ([]byte, error) {
  var d struct {
    I        int
    J        int
    Stations []string
    City     string
    Months   [12]byte
    Total    byte
  }

  d.I = r.I
  d.J = r.J
  d.City = r.City

  d.Months = r.Months
  d.Total = r.Total
  for _, station := range r.Nearest {
    d.Stations = append(d.Stations, station.Id())
  }

  return json.Marshal(&d)
}

// Convert a region and a years worth of data to a RegionStats object.
func toRegionStats(region *Region, data [12]Pct) *RegionStats {
  var r [12]byte
  var p Pct
  for i := 0; i < 12; i++ {
    r[i] = data[i].Byte()
    p.A += data[i].A
    p.B += data[i].B
  }

  return &RegionStats{
    Region: region,
    Months: r,
    Total:  p.Byte(),
  }
}

// Provides constant time lookup of RegionStats by (i,j)
type RegionStatsMap map[int]*RegionStats

// Create a new, empty RegionStatsMap
func NewRegionStatsMap() RegionStatsMap {
  return RegionStatsMap(map[int]*RegionStats{})
}

// Put the RegionStats object into the map.
func (r RegionStatsMap) Put(s *RegionStats) {
  r[s.I<<16|s.J] = s
}

// Find the RegionStats for the region (i,j)
func (r RegionStatsMap) Get(i, j int) *RegionStats {
  return r[i<<16|j]
}

// Most of the work will be done here as this computes that data for and writes the
// stats files for each region.
func WriteStatsFiles(dir string, store *gsod.Store, grid *Grid, tp *TempPref) error {
  m := map[string][][12]Pct{}
  for _, station := range store.Stations {
    m[station.Id()] = make([][12]Pct, len(store.Years))
  }

  for i, year := range store.Years {
    fmt.Printf("%d\n", year)
    if err := store.ForEachSummaryInYear(year, func(s *gsod.Summary) error {
      month := s.Day.Month() - 1
      p := &m[s.Station.Id()][i][month]
      if IsPleasant(s, tp) {
        p.A++
      }
      p.B++
      return nil
    }); err != nil {
      return err
    }
  }

  rm := NewRegionStatsMap()
  var overall []*RegionStats

  for i := 0; i < grid.W; i++ {
    for j := 0; j < grid.H; j++ {
      r := grid.Grid[i][j]
      if r == nil {
        continue
      }

      var allYears [12]Pct
      for y := 0; y < len(store.Years); y++ {
        var thisYear [12]Pct
        for _, station := range r.Nearest {
          s := m[station.Id()]
          for m := 0; m < 12; m++ {
            thisYear[m].A += s[y][m].A
            thisYear[m].B += s[y][m].B

            allYears[m].A += s[y][m].A
            allYears[m].B += s[y][m].B
          }
        }
      }

      rs := toRegionStats(r, allYears)
      rm.Put(rs)
      overall = append(overall, rs)
    }
  }

  hackKeyLargo(rm)

  util.Sort(len(overall),
    func(i, j int) bool {
      return overall[i].Total > overall[j].Total
    }, func(i, j int) {
      overall[i], overall[j] = overall[j], overall[i]
    })

  var s struct {
    W       int
    H       int
    Regions []*RegionStats
  }

  s.W = grid.W
  s.H = grid.H
  s.Regions = overall

  return WriteJson(filepath.Join(dir, fmt.Sprintf("%s.json", tp.Name)), &s)
}

// the data for key largo is jacked up, so just copy the data for long key
// to the south.
func hackKeyLargo(m RegionStatsMap) {
  kl := m.Get(77, 42)
  lk := m.Get(77, 43)
  kl.Months = lk.Months
  kl.Total = lk.Total
}

type Zip struct {
  Code string
  City string
  Pop  int
  Lat  float64
  Lon  float64
  X    float64
  Y    float64
}

// Load the zipcode data into the referenced array.
func LoadZips(filename string, zips *[]*Zip) error {
  r, err := os.Open(filename)
  if err != nil {
    return err
  }
  defer r.Close()

  if err := json.NewDecoder(r).Decode(zips); err != nil {
    return err
  }

  pops := map[string]int{}
  for _, zip := range *zips {
    pops[zip.City] += zip.Pop
  }

  for _, zip := range *zips {
    zip.Pop = pops[zip.City]
  }

  return nil
}

// Place the zips into the virtual coordinates using the transformation.
func PlaceZips(zips []*Zip, tx func(float64, float64) (float64, float64)) {
  for _, z := range zips {
    x, y := tx(z.Lat, z.Lon)
    z.X = x
    z.Y = y
  }
}

type zipIndexEntry struct {
  *Zip
  I int
  J int
}

// Custom marshaler to just pack everything into a JSON array.
func (z *zipIndexEntry) MarshalJSON() ([]byte, error) {
  a := [4]interface{}{
    z.Zip.Code,
    z.Zip.City,
    z.I,
    z.J,
  }
  return json.Marshal(a)
}

// Build an map of zipcode prefix strings to zipPrefix objects which contains
// the list of relevant zips associated with that prefix.
func indexByPrefix(zips []*zipIndexEntry, l int) map[string]*zipPrefix {
  idx := map[string]*zipPrefix{}
  for _, zip := range zips {
    p := zip.Code[:l]
    v := idx[p]
    if v == nil {
      v = &zipPrefix{}
      idx[p] = v
    }
    v.Z = append(v.Z, zip)
  }
  return idx
}

// Truncate the list of zip index entries to n items.
func limitBy(zips []*zipIndexEntry, n int) []*zipIndexEntry {
  if len(zips) < n {
    return zips
  }
  return zips[:n]
}

// All the data for a particular zip prefix.
type zipPrefix struct {
  // the data needed to offer N completions for this prefix
  Z []*zipIndexEntry
  // the coordinates of the regions to highlight on the map,
  // J is the lower 8 bits, I is the next 8 bits. This just
  // slims down the JSON structure a bit.
  C []int32
}

// Takes a zipPrefix that is populated with zips, computes
// the set of coordinates as C and then truncates the list
// of zips to N. If n is -1, do not truncate.
func (z *zipPrefix) build(n int) {
  m := map[int32]bool{}
  for _, zip := range z.Z {
    m[int32((zip.I<<8)|zip.J)] = true
  }

  for c, _ := range m {
    z.C = append(z.C, c)
  }

  if n >= 0 {
    z.Z = limitBy(z.Z, n)
  }
}

// Write the zip index files for the given prefix and then descend to longer
// prefixes writing those associated files.
func writeZipFiles(dir, prefix string, n int, zips []*zipIndexEntry) error {
  l := len(prefix) + 1
  m := indexByPrefix(zips, l)
  limit := -1

  if l < 4 {
    limit = n
    for prefix, zips := range m {
      if err := writeZipFiles(dir, prefix, n, zips.Z); err != nil {
        return err
      }
    }
  }

  for _, zips := range m {
    zips.build(limit)
  }

  if prefix != "" {
    dir = filepath.Join(dir, prefix[:1])
  }

  // make sure the diretory exists
  if _, err := os.Stat(dir); err != nil {
    if err := os.MkdirAll(dir, os.ModePerm); err != nil {
      return err
    }
  }

  name := "root.json"
  if l > 1 {
    name = fmt.Sprintf("%s.json", prefix)
  }

  return WriteJson(filepath.Join(dir, name), m)
}

// Generates all the files that make up the zip-code index. This is
// a heirarchical directory/file structure that allows auto-suggest
// to work with just static files.
func WriteZipIndex(dir string, grid *Grid, n int) error {
  var zips []*zipIndexEntry
  // build a list of zips
  for i := 0; i < grid.W; i++ {
    for j := 0; j < grid.H; j++ {
      r := grid.Grid[i][j]
      if r == nil {
        continue
      }

      for _, zip := range r.Zips {
        zips = append(zips, &zipIndexEntry{
          Zip: zip,
          I:   r.I,
          J:   r.J,
        })
      }
    }
  }

  // sort by descending population
  util.Sort(len(zips), func(i, j int) bool {
    return zips[j].Pop < zips[i].Pop
  }, func(i, j int) {
    zips[i], zips[j] = zips[j], zips[i]
  })

  return writeZipFiles(dir, "", n, zips)
}

// Just ensure that the directory exists.
func EnsureDir(path string) error {
  if _, err := os.Stat(path); err != nil {
    return os.MkdirAll(path, os.ModePerm)
  }
  return nil
}

// Write a json file with information about the grid. This is used for
// debugging assignment of stations and zips to regions.
func WriteGridInfoFile(filename string, grid *Grid) error {
  w, err := os.Create(filename)
  if err != nil {
    return err
  }
  defer w.Close()

  e := json.NewEncoder(w)

  for i := 0; i < grid.W; i++ {
    for j := 0; j < grid.H; j++ {
      r := grid.Grid[i][j]
      if r == nil {
        continue
      }

      m := map[string]interface{}{
        "City": r.City,
        "I":    r.I,
        "J":    r.J,
        "Zips": r.Zips,
      }

      if err := e.Encode(m); err != nil {
        return err
      }
    }
  }
  return nil
}

func main() {
  flagWork := flag.String("work", "work", "the destination work directory")
  flagData := flag.String("data", "data", "the source data directory")
  flag.Parse()

  var zips []*Zip
  if err := LoadZips(filepath.Join(*flagWork, "zips.json"), &zips); err != nil {
    panic(err)
  }

  store, err := gsod.OpenStore(*flagData)
  if err != nil {
    panic(err)
  }

  var c GridConfig
  if err := LoadGridConfig(filepath.Join(*flagData, "grid.json"), &c); err != nil {
    panic(err)
  }

  r := image.Rect(0, 0, 1024, 768)
  tx := ComputeTransform(store.Stations, r)

  PlaceZips(zips, tx)

  grid := BuildGrid(PlaceStations(store.Stations, tx), zips, r, &c)

  if err := WriteGridInfoFile(filepath.Join(*flagWork, "info.json"), grid); err != nil {
    panic(err)
  }

  for _, prefs := range TempPrefs {
    if err := WriteStatsFiles(*flagWork, store, grid, prefs); err != nil {
      panic(err)
    }
  }

  zipDir := filepath.Join(*flagWork, "z")
  if err := EnsureDir(zipDir); err != nil {
    panic(err)
  }

  if err := WriteZipIndex(zipDir, grid, 10); err != nil {
    panic(err)
  }
}
