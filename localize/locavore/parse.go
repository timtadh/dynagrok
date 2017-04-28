package locavore

import (
	"bufio"
	"github.com/timtadh/dynagrok/dgruntime/dgtypes"
	"io"
	"io/ioutil"
	"log"
	"strings"
)

// ParseProfiles takes two Readers, and parses and unserializes the profiles
// that come from each of them.
func ParseProfiles(ok io.Reader, fail io.Reader) ([]dgtypes.Type, []dgtypes.FuncProfile, []dgtypes.FuncProfile) {
	typesok, parsedOk := parseProfile(ok)
	typesfail, parsedFail := parseProfile(fail)

	types := append(dgtypes.UnserializeType(typesok).Types, dgtypes.UnserializeType(typesfail).Types...)
	return types, unserializeFuncs(parsedOk), unserializeFuncs(parsedFail)
}

// parseProfile returns a tuple containing the types defined
// on line 1 of the file, and a list of each funcProfile string
func parseProfile(r io.Reader) (string, []string) {
	p := bufio.NewReader(r)
	types, err := p.ReadString('\n')
	if err != nil {
		log.Panic("Locavore: Error reading file")
	}
	content, err := ioutil.ReadAll(p)
	if err != nil {
		log.Panic("Locavore: Error reading file")
	}
	return types, strings.Split(strings.TrimSpace(string(content)), "\n")
}

// Unserializes each funcprofile
func unserializeFuncs(profiles []string) (profs []dgtypes.FuncProfile) {
	for _, s := range profiles {
		object := dgtypes.UnserializeFunc(s)
		profs = append(profs, object)
	}
	//log.Printf("%d profs: \n\t %v", len(profs), profs)
	return profs
}
