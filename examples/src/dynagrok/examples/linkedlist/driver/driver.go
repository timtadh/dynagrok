package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
)

const (
	profilePath = "/tmp/dynagrok-profile/object-profiles.json"
	failPath    = "fail"
	passPath    = "pass"
	numtests    = 1000
)

type Test string

func main() {
	tests := constructTests()
	if len(os.Args) != 3 {
		fmt.Printf("Command:\n\tdriver <system under test> <oracle>")
		os.Exit(1)
	}

	log.Printf("Clearing %v and %v...", failPath, passPath)
	ioutil.WriteFile(failPath, []byte{}, 0600)
	ioutil.WriteFile(passPath, []byte{}, 0600)

	sut := os.Args[1]
	oracle := os.Args[2]

	log.Printf("Running <%v> tests and checking output...", numtests)
	for i := 0; i < numtests; i++ {
		out1, err1 := runTest(sut, tests[i])
		out2, err2 := runTest(oracle, tests[i])
		check(out1, err1, out2, err2)
	}
	log.Printf("Tests completed, profiles have been placed accordingly")
}

func runTest(program string, test Test) (string, error) {
	cmd := exec.Command(program)
	stdin, err := cmd.StdinPipe()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic("")
	}
	io.WriteString(stdin, string(test))

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	defer writer.Flush()

	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	go io.Copy(writer, stdout)
	cmd.Wait()

	return string(buf.Bytes()), err
}

func check(out1 string, err1 error, out2 string, err2 error) {
	if !(err1 == nil && err2 == nil || err1 != nil && err2 != nil) {
		panic("A panic could have interrupted a function")
	} else if out1 != out2 {
		profs, err := ioutil.ReadFile(profilePath)
		f, err := os.OpenFile(failPath, os.O_APPEND|os.O_WRONLY, 0600)
		if _, err2 := f.Write(profs); err != nil || err2 != nil {
			panic("File not found, or write issue")
		}
	} else {
		profs, err := ioutil.ReadFile(profilePath)
		f, err := os.OpenFile(passPath, os.O_APPEND|os.O_WRONLY, 0600)
		if _, err2 := f.Write(profs); err != nil || err2 != nil {
			panic("File not found, or write issue")
		}
	}
}

func constructTests() []Test {
	tests := make([]Test, numtests)

	rand.Seed(22)
	cmds := [4]string{"put", "has", "show", "pop"}
	for i := 0; i < numtests; i++ {
		test := ""
		length := rand.Intn(10)
		for j := 0; j < length; j++ {
			cmd := ""
			arg := fmt.Sprintf("%d", rand.Intn(length))
			index := rand.Intn(2)
			switch index {
			case 0:
				cmd = cmds[index] + " " + arg
			case 1:
				cmd = cmds[index] + " " + arg
			case 2:
				cmd = cmds[index]
			case 3:
				cmd = cmds[index]
			default:
				panic("")
			}
			test = test + cmd + "\n"
		}

		test = test + "exit\n"
		//ioutil.WriteFile(fmt.Sprintf("test%d", i), []byte(test), 0600)
		tests[i] = Test(test)
	}
	return tests
}
