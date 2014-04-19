package util

import "time"

func YearInfo(yr int) (time.Time, int) {
  a := time.Date(yr, time.January, 1, 0, 0, 0, 0, time.UTC)
  z := time.Date(yr+1, time.January, 1, 0, 0, 0, 0, time.UTC)
  return a, int(z.Sub(a).Hours() / 24)
}
