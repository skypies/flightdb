package ui

import(
	"errors"
	"html/template"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/skypies/util/date"
)

func TemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"add": templateAdd,
		"flatten": templateFlatten,
		"sort": templateSort,                 // <p value="{{sort .AStringSlice | flatten}}" />
		"dict": templateDict,                 // {{template "foo" dict "Key" "Val" "OtherArgs" . }}
		"unprefixdict": templateUnprefixDict, // {{template "foo" unprefixdict "foo_prefix" . }}
		"nlldict": templateExtractNLLParams,  // only used by the widget-waypoint-or-pos template
		"selectdict": templateSelectDict,
		"km2feet": templateKM2Feet,
		"spacify": templateSpacifyFlightNumber,
		"formatPdt": templateFormatPdt,
	}
}


func templateAdd(a int, b int) int { return a + b }
func templateKM2Feet(x float64) float64 { return x * 3280.84 }
func templateSpacifyFlightNumber(s string) string {
	s2 := regexp.MustCompile("^r:(.+)$").ReplaceAllString(s, "Registration:$1")
	s3 := regexp.MustCompile("^(..)(\\d\\d\\d)$").ReplaceAllString(s2, "$1 $2")
	return regexp.MustCompile("^(..)(\\d\\d)$").ReplaceAllString(s3, "$1  $2")
}
func templateFlatten(in []string) string { return strings.Join(in, " ") }	
func templateSort(in []string) []string {
	sort.Strings(in)
	return in
}	

func templateFormatPdt(t time.Time, format string) string {
	return date.InPdt(t).Format(format)
}

// Args are treated as a sequence of keys and vals, and built into a map. Used to let you
// specify parameters for a sub-template.
func templateDict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 { return nil, errors.New("invalid dict call")	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i+=2 {
		key, ok := values[i].(string)
		if !ok { return nil, errors.New("dict keys must be strings") }
		dict[key] = values[i+1]
	}
	return dict, nil
}

// First arg is a prefix. Second arg is a map. Result is a map that contains just those keyval
// pairs whose key starts with the prefix; the prefix itself (plus '_' separator) is removed.
func templateUnprefixDict(prefix string, valueMap interface{}) map[string]interface{} {
	dict := map[string]interface{}{}
	for k,v := range valueMap.(map[string]interface{}) {
		strs := regexp.MustCompile("^"+prefix+"_(.*)$").FindStringSubmatch(k)
		if len(strs) < 2 {
			continue
		}
		dict[strs[1]] = v
	}
	return dict
}

// This comes from complaints. Template functions are a mess right now :(
func templateSelectDict(name, dflt string, vals interface{}) map[string]interface{} {
	return map[string]interface{}{
		"Name": name,
		"Default": dflt,
		"Vals": vals,
	}
}

// Returns a dict containing all the paramters needed to render the waypoint-or-pos widget template.
// Pulls three default values out of the valueMap, which mustt be prefixed by stem, and rewrites
// their keys as s/$stem_(.*)/nll_$1/
func templateExtractNLLParams(stem string, valueMap interface{}) map[string]interface{} {
	in := valueMap.(map[string]interface{})

	out := map[string]interface{}{
		"nll_stem": stem,
		"nll_waypoints": in["Waypoints"],
		"nll_name":  in[stem+"_name"],
		"nll_lat":   in[stem+"_lat"],
		"nll_long":  in[stem+"_long"],
	}

	return out
}
