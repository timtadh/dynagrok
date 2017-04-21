package locavore

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"github.com/timtadh/dynagrok/dgruntime/dgtypes"
)

func ParseProfiles(ok io.Reader, fail io.Reader) ([]dgtypes.Type, []dgtypes.FuncProfile, []dgtypes.FuncProfile) {
	typesok, parsedOk := parseProfile(ok)
	typesfail, parsedFail := parseProfile(fail)

	types := append(dgtypes.UnserializeType(typesok).Types, dgtypes.UnserializeType(typesfail).Types...)
	return types, unserializeFuncs(parsedOk), unserializeFuncs(parsedFail)
}

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

func unserializeFuncs(profiles []string) (profs []dgtypes.FuncProfile) {
	for _, s := range profiles {
		object := dgtypes.UnserializeFunc(s)
		profs = append(profs, object)
	}
	//log.Printf("%d profs: \n\t %v", len(profs), profs)
	return profs
}
