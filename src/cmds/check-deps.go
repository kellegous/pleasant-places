package main

import (
  "bufio"
  "bytes"
  "fmt"
  "io"
  "os"
  "os/exec"
  "regexp"
  "strconv"
  "strings"
)

var (
  nodeVersion = []int{0, 8, 0}
  tscVersion  = []int{0, 9, 5}
  sassVersion = []int{3, 2, 12}
)

func readLines(r io.Reader) ([]string, error) {
  var lines []string
  var buf bytes.Buffer
  br := bufio.NewReader(r)

  for {
    b, p, err := br.ReadLine()
    if err == io.EOF {
      return lines, nil
    } else if err != nil {
      return nil, err
    }

    buf.Write(b)
    if p {
      continue
    }

    lines = append(lines, buf.String())
    buf.Reset()
  }
}

func versionOk(has string, needs []int) bool {
  p := strings.Split(has, ".")
  h := make([]int, len(needs))
  for i, v := range p {
    j, err := strconv.ParseInt(v, 10, 64)
    if err != nil {
      return false
    }

    h[i] = int(j)
  }

  for i, n := 0, len(h); i < n; i++ {
    if h[i] > needs[i] {
      return true
    }

    if h[i] < needs[i] {
      return false
    }
  }

  return true
}

func findSubmatchInLines(p *regexp.Regexp, lines []string) []string {
  for _, line := range lines {
    if m := p.FindStringSubmatch(line); len(m) > 0 {
      return m[1:]
    }
  }
  return nil
}

func run(name string, args ...string) ([]string, error) {
  c := exec.Command(name, args...)

  r, err := c.StdoutPipe()
  if err != nil {
    return nil, err
  }
  defer r.Close()

  if err := c.Start(); err != nil {
    return nil, err
  }

  return readLines(r)
}

func check(cmd []string, pat *regexp.Regexp, needs []int) bool {
  lines, err := run(cmd[0], cmd[1:]...)
  if err != nil {
    return false
  }

  m := findSubmatchInLines(pat, lines)
  if m == nil {
    return false
  }

  return versionOk(m[0], needs)
}

func main() {
  hasTsc := check([]string{"tsc", "--version"},
    regexp.MustCompile("^Version (\\d+\\.\\d+\\.\\d+)"),
    tscVersion)

  hasNode := hasTsc || check([]string{"node", "--version"},
    regexp.MustCompile("^v(\\d+\\.\\d+\\.\\d+)"),
    nodeVersion)

  hasSass := check([]string{"sass", "--version"},
    regexp.MustCompile("^Sass (\\d+\\.\\d+\\.\\d+)"),
    sassVersion)

  if hasTsc && hasNode && hasSass {
    return
  }

  fmt.Println("I'm gonna let you finish, but you are missing some dependencies....\n")
  if !hasTsc {
    fmt.Println("Missing: TypeScript 0.9.5")
    if !hasNode {
      fmt.Println("  First, you need to install or upgrade node.js, visit http://nodejs.org/")
      fmt.Println("  Then, run: sudo npm install -g typescript")
    } else {
      fmt.Println("  Run: sudo npm install -g typescript")
    }
  }

  if !hasSass {
    if !hasTsc {
      fmt.Println()
    }

    fmt.Println("Missing: Sass 3.2.12")
    fmt.Println("  Run: sudo gem install sass")
  }

  os.Exit(1)
}
