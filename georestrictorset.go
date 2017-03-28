package flightdb

import(
	"bytes"
	"encoding/gob"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"
)

type RestrictorCombinationLogic int
const(
	CombinationLogicAll RestrictorCombinationLogic = iota
	CombinationLogicAny
)
func (l RestrictorCombinationLogic)String() string {
	return map[RestrictorCombinationLogic]string{
		CombinationLogicAll: "all",
		CombinationLogicAny: "any",
	}[l]
}

type GeoRestrictorSet struct {
	Name       string
	User       string
	Logic      RestrictorCombinationLogic
	Tags     []string

	R        []geo.Restrictor
	
	DSKey      string
}

// {{{ grs.String

func (grs GeoRestrictorSet)String() string {
	str := fmt.Sprintf("GeoRestrictorSet '%s' <%s>\n* Tags: [%s]\n", grs.Name, grs.User,
		strings.Join(grs.Tags, ","))
	str += fmt.Sprintf("%s of:-\n", grs.Logic)
	for i,gr := range grs.R {
		str += fmt.Sprintf("* [%02d] %s\n", i, gr)
	}
	return str
}

// }}}
// {{{ grs.OnelineString

func (grs GeoRestrictorSet)OnelineString() string {
	tagStr := ""
	if len(grs.Tags) > 0 { tagStr = "[" + strings.Join(grs.Tags, ",") + "]" }

	logicStr := ""
	if len(grs.R) > 1 { logicStr = strings.Title(grs.Logic.String()) + " of: " }
	
	str := fmt.Sprintf("%s%s, %s", grs.Name, tagStr, logicStr)
	grStrs := []string{}
	for _,gr := range grs.R {
		grStrs = append(grStrs, gr.String())
	}
	str += strings.Join(grStrs, "; ")
	return str
}

// }}}
// {{{ grs.Debug

func (grs GeoRestrictorSet)Debug() string {
	str := fmt.Sprintf("%s\n----{ DEBUG }----\n", grs)

	for i,_ := range grs.R {
		str += fmt.Sprintf("--{ %02d }--\n%s\n", i, grs.R[i].GetDebug())
	}
	return str
}

// }}}
// {{{ grs.IsNil, IsAdhoc

func (grs GeoRestrictorSet)IsNil() bool {
	return (grs.Name == "" && len(grs.R) == 0 && grs.DSKey == "")
}


func (grs GeoRestrictorSet)IsAdhoc() bool {
	return (grs.Name == "ad-hoc")
}

// }}}

// {{{ GeoRestrictorIntoParams

func GeoRestrictorIntoParams(gr geo.Restrictor, p map[string]interface{}) {
	widget.ValuesIntoTemplateParams("", GeoRestrictorAsValues(gr), p)
}

// }}}
// {{{ BlankGeoRestrictorIntoParams

// Also called externally
func BlankGeoRestrictorIntoParams(p map[string]interface{}) {
	grBlank := geo.SquareBoxRestriction{}	
	p["GR"] = grBlank
	GeoRestrictorIntoParams(grBlank, p)
	p["gr_type"] = "" // unknown at this time
}

// }}}
// {{{ FormValueGeoRestrictor

func FormValueGeoRestrictor(r *http.Request) (geo.Restrictor, error) {
	var gr geo.Restrictor
	
	switch r.FormValue("gr_type") {
	case "squarebox":
		gr = geo.SquareBoxRestriction{
			Debugger: new(geo.DebugLog),
			NamedLatlong: sfo.FormValueNamedLatlong(r, "sb_center"),
			SideKM: widget.FormValueFloat64EatErrs(r, "sb_sidekm"),
			AltitudeMin: widget.FormValueInt64(r, "sb_altmin"),
			AltitudeMax: widget.FormValueInt64(r, "sb_altmax"),
			IsExcluding: widget.FormValueCheckbox(r, "sb_isexcluding"),
		}

	case "verticalplane":
		gr = geo.VerticalPlaneRestriction{
			Debugger: new(geo.DebugLog),
			Start: sfo.FormValueNamedLatlong(r, "vp_start"),
			End: sfo.FormValueNamedLatlong(r, "vp_end"),
			AltitudeMin: widget.FormValueInt64(r, "vp_altmin"),
			AltitudeMax: widget.FormValueInt64(r, "vp_altmax"),
			IsExcluding: widget.FormValueCheckbox(r, "vp_isexcluding"),
		}

	case "polygon":
		poly := geo.NewPolygon()
		for i:=0; i<10; i++ {
			if namedPt := sfo.FormValueNamedLatlong(r, fmt.Sprintf("poly_p%d", i)); !namedPt.IsNil() {
				poly.AddPoint(namedPt.Latlong)
			}
		}
		gr = geo.PolygonRestriction{
			Debugger: new(geo.DebugLog),
			Polygon: poly,
			AltitudeMin: widget.FormValueInt64(r, "poly_altmin"),
			AltitudeMax: widget.FormValueInt64(r, "poly_altmax"),
			IsExcluding: widget.FormValueCheckbox(r, "poly_isexcluding"),
		}

	default:
		return nil, fmt.Errorf("formValueGeoRestrictor: Fell out of switch")
	}

	return gr, nil
}

// }}}
// {{{ GeoRestrictorAsValues

// TODO: Should figure out a nice way to get this into the geo.Restrictor interface, without
// making geo/ depend on util/widget
func GeoRestrictorAsValues(gr geo.Restrictor) url.Values {
	v := url.Values{}

	switch t := gr.(type) {
	case geo.SquareBoxRestriction:
		v.Set("gr_type", "squarebox")
		widget.AddPrefixedValues(v, t.NamedLatlong.Values(), "sb_center")
		v.Set("sb_sidekm", fmt.Sprintf("%.3f", t.SideKM))
		if t.IsExcluding {
			v.Set("sb_isexcluding", "1")
		} else {
			v.Set("sb_isexcluding", "")
		}
		v.Set("sb_altmin", fmt.Sprintf("%d", t.AltitudeMin))
		v.Set("sb_altmax", fmt.Sprintf("%d", t.AltitudeMax))

	case geo.VerticalPlaneRestriction:
		v.Set("gr_type", "verticalplane")
		widget.AddPrefixedValues(v, t.Start.Values(), "vp_start")
		widget.AddPrefixedValues(v, t.End.Values(), "vp_end")
		if t.IsExcluding {
			v.Set("vp_isexcluding", "1")
		} else {
			v.Set("vp_isexcluding", "")
		}
		v.Set("vp_altmin", fmt.Sprintf("%d", t.AltitudeMin))
		v.Set("vp_altmax", fmt.Sprintf("%d", t.AltitudeMax))

	case geo.PolygonRestriction:
		v.Set("gr_type", "polygon")
		for i,pos := range t.GetPoints() {
			widget.AddPrefixedValues(v, pos.Values(), fmt.Sprintf("poly_p%d", i+1))
		}
		if t.IsExcluding {
			v.Set("poly_isexcluding", "1")
		} else {
			v.Set("poly_isexcluding", "")
		}
		v.Set("poly_altmin", fmt.Sprintf("%d", t.AltitudeMin))
		v.Set("poly_altmax", fmt.Sprintf("%d", t.AltitudeMax))
	}

	return v
}

// }}}

// {{{ f.SatisfiesGeoRestrictorSet

// Kinda useless without the outcome intersections
func (f *Flight)SatisfiesGeoRestrictorSet(grs GeoRestrictorSet) (bool, RestrictorSetIntersectOutcome) {
	it := f.GetIntersectableTrack() // build once; a bit expensive
	outcome := it.SatisfiesRestrictorSet(grs)
	return outcome.Satisfies(grs.Logic), outcome
}

// }}}


// This is the object which is persisted into Datastore
type IndexedRestrictorSetBlob struct {
	Blob         []byte      `datastore:",noindex"`

	Name           string
	Tags         []string
	User           string
}

// {{{ grs.ToBlob

func (grs *GeoRestrictorSet)ToBlob() (*IndexedRestrictorSetBlob, error) {
	var buf bytes.Buffer

	_ = grs.Debug() // Drain any debug logs

	if err := gob.NewEncoder(&buf).Encode(grs); err != nil {
		return nil,err
	}

	sort.Strings(grs.Tags)
	
	return &IndexedRestrictorSetBlob{
		Blob: buf.Bytes(),
		Tags: grs.Tags,
		User: grs.User,
	}, nil
}

// }}}
// {{{ blob.ToRestrictorSet

func (blob *IndexedRestrictorSetBlob)ToRestrictorSet(key string) (*GeoRestrictorSet, error) {
	buf := bytes.NewBuffer(blob.Blob)
	grs := GeoRestrictorSet{}
	err := gob.NewDecoder(buf).Decode(&grs)

	grs.DSKey = key

	return &grs, err
}

// }}}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
