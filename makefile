DIR := ${CURDIR}
HOME := $(HOME)
PROG := dynagrok/examples/linkedlist

install:
	go install github.com/timtadh/dynagrok

clientbad:
	git -C examples/src/dynagrok/examples/shapes checkout dynagrokfaulty
	dynagrok -r ~/dev/repos/go-research -d ${HOME}/dynagrok/src/github.com/timtadh/dynagrok -g ${HOME}/dynagrok/src/github.com/timtadh/dynagrok/examples objectstate --keep-work -w /tmp/work dynagrok/examples/shapes/client
	git -C  examples/src/dynagrok/examples/shapes checkout dynagrok

clientgood:
	git -C examples/src/dynagrok/examples/shapes checkout dynagrok
	dynagrok -r ~/dev/repos/go-research -d ${HOME}/dynagrok/src/github.com/timtadh/dynagrok -g ${HOME}/dynagrok/src/github.com/timtadh/dynagrok/examples objectstate --keep-work -w /tmp/work dynagrok/examples/shapes/client

client:
	dynagrok -r ~/dev/repos/go-research -d ${HOME}/dynagrok/src/github.com/timtadh/dynagrok -g ${HOME}/dynagrok/src/github.com/timtadh/dynagrok/examples objectstate --keep-work -w /tmp/work dynagrok/examples/shapes/client

prog:
	dynagrok -r ~/dev/repos/go-research -d ${HOME}/dynagrok/src/github.com/timtadh/dynagrok -g ${HOME}/dynagrok/src/github.com/timtadh/dynagrok/examples objectstate --keep-work -w /tmp/work ${PROG}

method:
	dynagrok -r ~/dev/repos/go-research -d ${HOME}/dynagrok/src/github.com/timtadh/dynagrok -g ${HOME}/dynagrok/src/github.com/timtadh/dynagrok/examples objectstate -m Move --keep-work -w /tmp/work dynagrok/examples/shapes/client

clean:
	rm /tmp/work/goroot/src/dgruntime* -r
	rm /tmp/work/goroot/pkg/linux_amd64/dgruntime* -r
	#rm *.instr
