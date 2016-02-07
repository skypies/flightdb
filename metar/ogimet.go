package metar

/*

import(
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"regexp"
	"time"
)

// {{{ OgimetPreParse

// Strip out comment lines and blank lines; glue together continuation lines.
// Leave the timestamp prefix in place.

func OgimetPreParse(in string) []string {
	out := []string{}
	
	scanner := bufio.NewScanner(strings.NewReader(in))
	curr := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 { continue }
		if line[0] == '#' { continue }
		curr += line
		if curr[len(curr)-1] == '=' {
			out = append(out, curr)
			curr = ""
		} else {
			curr += " "
		}
	}

	return out
}

// }}}
// {{{ OgimetParse

// Ogimet has a full timestamp prefix:
// 201601070156 METAR KSFO 070156Z 16006KT 10SM -RA FEW019 SCT027 BKN049=

func OgimetParse(s string) (*Report, error) {
	s = strings.TrimSpace(s)

	ogimetPrefixR := regexp.MustCompile("^([0-9]{12}) (.*)$")

	match := ogimetPrefixR.FindStringSubmatch(s)
	if len(match) != 3 {
		return nil, fmt.Errorf("bad ogimet formatting in '%s' (%d)", s, len(match))
	}

	t,err := time.Parse("200601021504", match[1]) // Default to UTC
	if err != nil {
		return nil, fmt.Errorf("bad ogimet timestamp in '%s', %v", match[1], err)
	}

	return Parse(match[2], t)
}

// }}}

// {{{ FetchFromOgimet

// http://www.ogimet.com/display_metars2.php?lang=en&lugar=KSFO&tipo=SA&ord=REV&nil=NO&fmt=txt&ano=2016&mes=01&day=06&hora=01&anof=2016&mesf=01&dayf=07&horaf=01&minf=59&send=send

func FetchFromOgimet(c *http.Client, station string, s,e time.Time) (*Archive,error) {
	url := "http://www.ogimet.com/display_metars2.php?"+
		"lang=en&ord=REV&nil=NO&fmt=txt&send=send&tipo=SA"
	url += "&lugar="+station
	url += s.Format("&ano=2006&mes=01&day=02&hora=15")
	url += e.Format("&anof=2006&mesf=01&dayf=02&horaf=15&minf=04")

	if c == nil {
		client := http.Client{}
		c = &client
	}
	
	resp,err := c.Get(url)
	if err != nil { return nil, err }

	defer resp.Body.Close()
	body,err := ioutil.ReadAll(resp.Body)
	if err != nil { return nil, err }

	fmt.Printf("Wahwy !!\n----\n%s\n----\n", body)

	a := Archive{}
	return &a,nil
}

// }}}

*/

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
