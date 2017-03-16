package flightdb

// go test -v github.com/skypies/flightdb2

import "testing"

type CallsignTest struct {
	Raw        string
	Normalized string
	CallsignType
}

var tests = []CallsignTest{
	{"",         "",         JunkCallsign},
	{"-.-.-.-.", "-.-.-.-.", JunkCallsign},
	{"N761QA",   "N761QA",   Registration},
	{"UAL100",   "UAL100",   IcaoFlightNumber},
	{"987",      "987",      BareFlightNumber},
	{"VRD010",   "VRD10",    IcaoFlightNumber}, // Check zeroes get stripped
	{"SKW750R",  "SKW750",   IcaoFlightNumber}, // Check suffix get stripped
}

func TestParseCallsign(t *testing.T) {
	for _,test := range tests {
		cs := NewCallsign(test.Raw)
		if cs.CallsignType != test.CallsignType {
			t.Errorf("'%s' - expected type %v, got %v", test.Raw, test.CallsignType, cs.CallsignType)
		}
		if cs.String() != test.Normalized {
			t.Errorf("'%s' - expected string %q, got %q", test.Raw, test.Normalized, cs.String())
		}
	}
}
