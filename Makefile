DATA=data/gsod_1990.tar \
	data/gsod_1991.tar \
	data/gsod_1992.tar \
	data/gsod_1993.tar \
	data/gsod_1994.tar \
	data/gsod_1995.tar \
	data/gsod_1996.tar \
	data/gsod_1997.tar \
	data/gsod_1998.tar \
	data/gsod_1999.tar \
	data/gsod_2000.tar \
	data/gsod_2001.tar \
	data/gsod_2002.tar \
	data/gsod_2003.tar \
	data/gsod_2004.tar \
	data/gsod_2005.tar \
	data/gsod_2006.tar \
	data/gsod_2007.tar \
	data/gsod_2008.tar \
	data/gsod_2009.tar \
	data/gsod_2010.tar \
	data/gsod_2011.tar \
	data/gsod_2012.tar \
	data/gsod_2013.tar

ALL : work/norm.json

src/github.com/kellegous/pork:
	@GOPATH=`pwd` go get github.com/kellegous/pork

src/github.com/ungerik/go-cairo:
	@GOPATH=`pwd` go get github.com/ungerik/go-cairo

bin/% : src/cmds/%.go
	@GOPATH=`pwd` go build -o $@ $<

bin/build-grid: src/github.com/ungerik/go-cairo work/zips.json
	@GOPATH=`pwd` go build -o $@ src/cmds/build-grid.go

data/gsod_%.tar : bin/download
	@./bin/download 1990-2013

work/zips.json: bin/build-zips
	@./bin/build-zips

work/norm.json: bin/build-grid
	@./bin/build-grid

clean:
	rm -rf work bin

nuke: clean
	rm -rf $(DATA) src/github.com