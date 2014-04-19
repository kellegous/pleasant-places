/// <reference path="lib/jquery.d.ts" />
/// <reference path="lib/signal.ts" />
module app {

var $c = (name : string, ns? : string) => {
  return ns ? $(document.createElementNS(ns, name)) : $(document.createElement(name));
};

var attr = (e : Element, n : string, v : any) => {
  e.setAttribute(n, '' + v);
};

var on = (e : Element, t : string, f : (e : Event) => void) => {
  e.addEventListener(t, f, false);
};

var SVGNS = 'http://www.w3.org/2000/svg',
    // TODO(knorton): Fix this.
    // BASEURL = location.pathname.substring(2),
    BASEURL = '/data/',
    MONTHS = ['J', 'F', 'M', 'A', 'M', 'J', 'J', 'A', 'S', 'O', 'N', 'D'],
    TRANSFORMS = ['-webkit-transform', '-moz-transform', '-ms-transform', '-o-transform', 'transform'];

interface Region {
  I : number;
  J : number;
  Stations : string[];
  City : string;
  Months : number[];
  Total : number;
}

interface Grid {
  W : number;
  H : number;
  Regions : Region[];
}

interface ZipPref {
  Z : any[];
  C : number[];
}

var UnpackPoints = (p : number[]) => {
  return p.map<number[]>((v : number) => {
    return [v >> 8, v & 0xff];
  });
};

var Transform = (e : JQuery, tx : number, r : number) => {
  var v = 'translateX(' + tx + 'px) rotate(' + r + 'deg)';
  TRANSFORMS.forEach((t) => {
    e.css(t, v);
  });
};

class Model {
  gridDidLoad = new Q.Signal;
  zipsDidLoad = new Q.Signal;

  grid : Grid;

  private zips = {};
  private pending = {};
  private prefixes = {};

  load() {
    window['model'] = this;
    $.getJSON(BASEURL + 'norm.json', (grid : Grid) => {
      this.grid = grid;
      this.gridDidLoad.raise(this);
    });

    $.getJSON(BASEURL + 'z/root.json', (zips : { [index : string] : ZipPref }) => {
      this.zips = zips;
      this.prefixes[''] = true;
      this.zipsDidLoad.raise(this);
    });
  }

  private fetchZips(pfx : string, cb : () => void) {
    var pending = this.pending,
        queue = pending[pfx],
        zips = this.zips,
        prefixes = this.prefixes,
        uri = 'z/' + pfx.substring(0, 1) + '/' + pfx + '.json';

    if (!queue) {
      queue = pending[pfx] = [];
      $.getJSON(BASEURL + uri, (data : { [index : string] : ZipPref; }) => {

        for (var key in data || {}) {
          var val = data[key];
          zips[key] = val;
          val.Z.forEach((v : any[]) => {
            zips[<string>v[0]] = v;
          });
        }

        prefixes[pfx] = true;
        pending[pfx] = null;

        queue.forEach((f : () => void) => {
          f();
        });

      });
    }

    queue.push(cb);
  }

  findZipCoords(zip : string, cb : (i : number, j : number, ok : boolean) => void) {
    var zips = this.zips,
        prefixes = this.prefixes;

    if (zip.length != 5) {
      cb(-1, -1, false);
      return;
    }

    var pfx = zip.substring(0, 3);
    if (prefixes[pfx]) {
      var v = zips[zip],
          i = v ? <number>v[2] : -1,
          j = v ? <number>v[3] : -1,
          k = v != null;
      cb(i, j, k);
      return;
    }

    this.fetchZips(pfx, () => {
      this.findZipCoords(zip, cb);
    });
  }

  suggestZipsFor(val : string, cb : (vals : any, coords : number[][]) => void) {
    var pfx = val.substring(0, val.length - 1),
        prefixes = this.prefixes,
        pending = this.pending,
        zips = this.zips,
        dispatch = (z : ZipPref) => {
          if (!z) {
            cb(null, null);
          } else {
            cb(z.Z, UnpackPoints(z.C || []));
          }
        };

    // if loaded, fire now.
    if (prefixes[pfx]) {
      dispatch(zips[val]);
    } else {
      // otherwise, load it on demand.
      this.fetchZips(pfx, () => {
        dispatch(zips[val]);
      });
    }

    // prefetch the next level of completion if there is one.
    if (val.length <= 3) {
      this.fetchZips(val, () => {
      });
    }
  }

}

class RegView {
  elemA : Element;
  elemB : Element;
  constructor(public region : Region) {
  }
}

class CallView {
  private showing : RegView;

  private timer : number;

  constructor(private root : JQuery,
    private text : JQuery,
    private graf : JQuery,
    private mons : JQuery) {
  }

  static build() : CallView {
    var root = $('#call'),
        text = $('.text', root).text('???'),
        graf = $('.graf', root),
        labs = $('.labs', root),
        mons = [];

    for (var i = 0; i < 12; i++) {
      $c('div').addClass('mnbg')
        .css('left', i * 20)
        .appendTo(graf);

      mons.push($c('div')
        .addClass('mnfg')
        .css('left', i * 20)
        .appendTo(graf).get(0));

      $c('div').addClass('mnlb')
        .css('left', i * 20)
        .text(MONTHS[i])
        .appendTo(labs);
    }

    return new CallView(root, text, graf, $(mons));
  }

  public scrollTo() {
    this.root.get(0).scrollIntoView();
  }

  private reset() {
    var showing = this.showing,
        timer = this.timer;

    if (showing) {
      attr(showing.elemA, 'class', 'node');
      this.showing = null;
    }

    if (timer >= 0) {
      clearTimeout(timer);
      this.timer = -1;
    }
  }

  showOn(rv : RegView) {
    this.reset();
    var root = this.root,
        text = this.text,
        elem = rv.elemA,
        mons = this.mons;

    root.show();

    var elemRect = elem.getBoundingClientRect(),
        rootRect = root.get(0).getBoundingClientRect(),
        city = rv.region.City || 'MIDDLE OF NOWHERE',
        total = rv.region.Total,
        gutter = 20;

    // compute where the left should be, then limit it to the sides of the screen.
    var al = elemRect.left + elemRect.width/2 - rootRect.width/2 + document.body.scrollLeft,
        rl = Math.max(gutter, Math.min(window.innerWidth + document.documentElement.scrollLeft - rootRect.width - gutter, al));

    // adjust the pointer by the distance we were unable to travel.
    Transform($('.nib>div', root), al - rl, 45);

    attr(elem, 'class', 'node sel');
    root.css('left', rl)
      .css('top', elemRect.top + $(document).scrollTop() - rootRect.height - 10);
    text.text(city).append($c('span').text(total + ' days/yr'));
    rv.region.Months.forEach((v : number, i : number) => {
      $(mons.get(i)).css('height', 50 * (v/255));
    });
    this.showing = rv;
  }

  hide(now? : boolean) {
    this.reset();

    if (now) {
      this.showing = null;
      this.root.hide();
      return;
    }

    this.timer = setTimeout(() => {
      this.hide(true);
    }, 500);
  }
}

interface Zip {
  code : string;
  i : number;
  j : number;
}

class Search {
  didFind = new Q.Signal;
  didRefine = new Q.Signal;
  didClear = new Q.Signal;
  wantsToSearch = new Q.Signal;
  didChangeState = new Q.Signal;

  active = false;

  private root : JQuery;
  private text : JQuery;
  private list : JQuery;

  private sugs : JQuery;
  private zips : JQuery;
  private city : JQuery;

  private current : string;
  private selected = -1;

  constructor(private model : Model, n? : number) {
    var sugs = [],
        zips = [],
        city = [],
        root = $('#zips'),
        text = $('input', root).prop('disabled', true),
        list = $c('ol').hide()
          .append($c('div').addClass('nib').append($c('div')))
          .appendTo(root);

    n = n || 10;
    for (var i = 0; i < n; i++) {
      var sug = $c('li');
      ((i : number) => {
        sug.on('mousedown', (e : MouseEvent) => {
          e.preventDefault();
        }).on('click', (e : MouseEvent) => {
          this.commit(i);
        });
      })(i);

      zips.push($c('span')
        .addClass('zip')
        .appendTo(sug));

      city.push($c('span')
        .addClass('city')
        .appendTo(sug));

      sugs.push(sug.appendTo(list));
    }

    model.zipsDidLoad.tap((model? : Model) => {
      text.prop('disabled', false);
    });

    text.on('keydown', (e : KeyboardEvent) => {
      var n = this.sugs.length;
      switch (e.keyCode) {
      case 40: // down
        this.select(Math.min(n - 1, Math.max(0, this.selected + 1)));
        break;
      case 38: // up
        this.select(Math.min(n - 1, Math.max(0, this.selected - 1)));
        break;
      case 27: // esc
        this.clear();
        // Firefox: escape will propagate and end up restoring the field to
        // it's previous value.
        e.stopPropagation();
        e.preventDefault();
        break;
      case 13: // enter
        if (this.selected != -1) {
          this.commit(this.selected);
        }
        break;
      }
    }).on('keypress', (e : KeyboardEvent) => {
      var cc = e.charCode;

      // firefox dispatches with bullshit charCodes when the key is not
      // printable.
      if (cc == 0 || e.ctrlKey || e.metaKey) {
        return;
      }

      // first line defense against entering non-digits, this prevents
      // the non-digits from showing up at all. The more general catch
      // is in update where non-digits are replaced.
      if (cc < 48 || cc > 57) {
        e.preventDefault();
      }
    }).on('keyup', (e : KeyboardEvent) => {
      this.update();
    }).on('change', (e : Event) => {
      console.log('change');
      this.update();
    }).on('paste', (e : Event) => {
      setTimeout(() => {
        this.update();
      }, 0);
    }).on('click', (e : Event) => {
      this.update();
    }).on('focus', () => {
      var val = text.val();
      // reactivate if there is text in the search box.
      if (val.length > 0) {
        this.activate(true);
        if (val.length < 5) {
          this.show();
        }
        this.update(true);
      }
    }).on('blur', () => {
      var val = text.val();
      this.hide();
      if (val.length < 5) {
        this.activate(false);
      }
    }).on('mouseover', (e : Event) => {
      if (!this.active) {
        this.wantsToSearch.raise();
      }
    });

    this.root = root;
    this.text = text;
    this.list = list;
    this.sugs = $(sugs);
    this.zips = $(zips);
    this.city = $(city);
  }

  clear() {
    this.select(-1);
    this.text.val('')
      .removeClass('error');
    this.update();
  }

  private activate(active : boolean) {
    if (this.active == active) {
      return;
    }

    this.active = active;
    if (active) {
      this.text.addClass('active');
    } else {
      this.text.removeClass('active');
    }

    this.didChangeState.raise(this.active);
  }

  private hide() {
    this.list.hide();
  }

  private show() {
    this.list.show();
  }

  private reset() {
    var sel = this.selected,
        sugs = this.sugs;
    if (sel != -1) {
      sugs.get(sel).removeClass('sel');
    }
    this.selected = -1;
    this.hide();
  }

  private commit(index : number) {
    this.text.val(this.sugs.get(index).attr('data-zip'));
    this.update();
  }

  private select(index : number) {
    var n = this.sugs.length,
        selected = this.selected,
        sugs = this.sugs;
    if (selected >= 0) {
      sugs.get(selected).removeClass('sel');
    }

    this.selected = index;
    if (index == -1) {
      return;
    }

    sugs.get(index).addClass('sel');
  }

  private update(force? : boolean) {
    var model = this.model,
        text = this.text.val(),
        sugs = this.sugs,
        zips = this.zips,
        city = this.city,
        list = this.list,
        n = sugs.length;

    // remove any non-digits that got added
    var clean = text.replace(/\D/g, '');
    if (text != clean) {
      this.text.val(clean);
      return;
    }

    if (!force && this.current == text) {
      return;
    }
    this.current = text;

    // search is active if there is any text.
    this.activate(text.length > 0);

    if (!text || text.length == 0) {
      this.reset();
      this.didClear.raise();
      return;
    } else if (text.length == 5) {
      this.reset();
      this.model.findZipCoords(text, (i : number, j : number, ok : boolean) => {
        if (!ok) {
          // TODO(knorton): I may not need this.
          console.log('not found', text);
          return;
        }
        this.didFind.raise(text, i, j);
      });
      return;
    }

    model.suggestZipsFor(text, (vals : any[][], coords : number[][]) => {
      if (!vals) {
        this.hide();
        this.text.addClass('error');
        return;
      }

      this.text.removeClass('error');

      // TODO(knorton): move the selection if it points at a hidden item.
      this.show();
      for (var i = 0; i < n; i++) {
        var val = vals[i];
        if (!val) {
          sugs.get(i).hide();
        } else {
          sugs.get(i).attr('data-zip', <string>val[0])
            .show();
          zips.get(i).text(<string>val[0]);
          city.get(i).text(<string>val[1]);
        }
      }

      this.didRefine.raise(text, vals.map<Zip>((v : any[]) => {
        return { code: <string>v[0], i: <number>v[2], j: <number>v[3] };
      }), coords);
    });
  }
}

class View {
  private root : JQuery;

  private call : CallView;

  private regsByIdx : RegView[] = [];

  private regsByCoord : RegView[] = [];

  private search : Search;

  private highlighted : JQuery;

  constructor(private model : Model) {
    this.root = $('#root');
    this.call = CallView.build();
    model.gridDidLoad.tap((model? : Model) => {
      this.build();
    });

    var search = this.search = new Search(model);

    search.didFind.tap((zip? : string, i? : number, j? : number) => {
      var grid = this.model.grid,
          regs = this.regsByCoord;
      this.highlight([]);
      this.call.showOn(regs[j*grid.W + i]);
    });
    search.didRefine.tap((zip? : string, res? : Zip[], coords? : number[][]) => {
      this.call.hide(true /*now*/);
      this.highlight(coords);
    });
    search.didClear.tap(() => {
      this.call.hide(true /*now*/);
      this.highlight([]);
    });
    search.wantsToSearch.tap(() => {
      this.call.hide(true);
    });
    search.didChangeState.tap((active? : boolean) => {
      this.highlight([]);
    });

    $(document.body).on('keydown', (e : KeyboardEvent) => {
      if (e.keyCode == 27 /* esc */) {
        this.call.hide(true);
        this.search.clear();
      }
    });

    $(window).on('resize', (e : Event) => {
      this.rebuild();
    });
  }

  private regionWasHovered(rv : RegView, over : boolean) {
    var call = this.call;

    // disable the hover if the search is active
    if (this.search.active) {
      return;
    }

    // hide on mouseout
    if (!over) {
      call.hide();
      return;
    }

    // show on mouseover
    call.showOn(rv);
  }

  private highlight(coords : number[][]) {
    var root = this.root,
        regs = this.regsByCoord,
        highlighted = this.highlighted,
        w = this.model.grid.W;

    if (highlighted) {
      highlighted.attr('class', 'node');
    }

    if (!coords.length) {
      root.attr('class', '');
      return;
    }

    root.attr('class', 'search');
    highlighted = $(coords.map((pt : number[]) => {
      return regs[pt[1]*w + pt[0]].elemA;
    }));

    this.highlighted = highlighted.attr('class', 'node hi');
  }

  // TODO(knorton): Make this work for mobile cases.
  private regionWasClicked(rv : RegView) {
  }

  public show(i : number, j : number) {
    this.search.clear();
    var rv = this.regsByCoord[j*this.model.grid.W + i];
    if (rv) {
      this.regionWasHovered(rv, true);
    }
  }

  public hide() {
    this.regionWasHovered(null, false);
  }

  private rebuild() {
    this.root.text('');
    this.regsByIdx = [];
    this.regsByCoord = [];
    // TODO(knorton): Resize will lose all the highlighted.
    this.highlighted = null;
    this.build();
  }

  private build() {
    var grid = this.model.grid,
        root = this.root,
        cont = root.parent(),
        w = Math.max(900, window.innerWidth - 120),
        dx = w / grid.W,
        h = dx * grid.Regions.reduce((m : number, r : Region) => {
          return Math.max(m, r.J + 1);
        }, 0),
        pad = 1;

    root.attr('width', w)
      .attr('height', h)
      .css('margin-left', (cont.get(0).offsetWidth - w) / 2);

    grid.Regions.forEach((region : Region) => {
      var v = region.Total>>5,
          r = dx/2,
          days = ((region.Total/255) * 356) | 0,
          x = dx*region.I,
          y = dx*region.J,
          rv = new RegView(region),
          onClicked = (e : Event) => {
            this.regionWasClicked(rv);
          },
          onMouseOver = (e : Event) => {
            this.regionWasHovered(rv, true);
          },
          onMouseOut = (e : Event) => {
            this.regionWasHovered(rv, false);
          };

      this.regsByIdx.push(rv);
      this.regsByCoord[region.J*grid.W + region.I] = rv;

      var bg = document.createElementNS(SVGNS, 'rect');
      attr(bg, 'x', x);
      attr(bg, 'y', y);
      attr(bg, 'width', 2*r);
      attr(bg, 'height', 2*r);
      attr(bg, 'fill', '#fff');
      attr(bg, 'stroke', 'none');
      on(bg, 'click', onClicked);
      on(bg, 'mouseover', onMouseOver);
      on(bg, 'mouseout', onMouseOut);
      root.get(0).appendChild(bg);

      var ea = document.createElementNS(SVGNS, 'circle');
      attr(ea, 'class', 'node');
      attr(ea, 'cx', x + r);
      attr(ea, 'cy', y + r);
      attr(ea, 'r', r - pad);
      root.get(0).appendChild(ea);
      rv.elemA = ea;

      if (v == 7) {
        return;
      }

      var eb = document.createElementNS(SVGNS, 'circle');
      attr(eb, 'cx', x + r);
      attr(eb, 'cy', y + r);
      attr(eb, 'r', 0.9 * (r - pad) * (1 - (v/7)));
      attr(eb, 'fill', '#fff');
      attr(eb, 'stroke', 'none');
      root.get(0).appendChild(eb);
      rv.elemB = eb;
    });
  }
}

var model = new Model,
    view = new View(model);

model.load();

var Show = (e : JQuery, scroll : boolean) => {
};

$('.regions>li').hover(
  function(e : Event) {
    var data = $(this).attr('data');
    if (!data) {
      return;
    }
    var pts = data.split(',');
    view.show(parseInt(pts[0]), parseInt(pts[1]));
  },
  function(e : Event) {
    view.hide();
  });

}
