package locavore

import (
	"github.com/timtadh/dynagrok/dgruntime/dgtypes"
	"io"
	"io/ioutil"
	"log"
	"strings"
)

func ParseProfiles(ok io.Reader, fail io.Reader) ([]dgtypes.FuncProfile, []dgtypes.FuncProfile) {
	parsedOk := parseProfile(ok)
	parsedFail := parseProfile(fail)
	return unserialize(parsedOk), unserialize(parsedFail)
}

func parseProfile(p io.Reader) []string {
	content, err := ioutil.ReadAll(p)
	if err != nil {
		log.Panic("Locavore: Error reading file")
	}
	return strings.Split(string(content), "\n")
}

func unserialize(profiles []string) (profs []dgtypes.FuncProfile) {
	for _, s := range profiles {
		profs = append(profs, dgtypes.UnserializeFunc(s))
	}
	return profs
}
