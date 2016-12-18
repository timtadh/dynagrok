DIR := ${CURDIR}

install:
	go install github.com/timtadh/dynagrok

clientbad:
	git -C examples/src/dynagrok/examples/shapes checkout dynagrokfaulty
	dynagrok -g $(DIR)/examples -d $(DIR) instrument -w /tmp/work --keep-work -o clientbad.instr dynagrok/examples/shapes/client
	git -C  examples/src/dynagrok/examples/shapes checkout dynagrok

clientgood:
	git -C examples/src/dynagrok/examples/shapes checkout dynagrok
	dynagrok -g $(DIR)/examples -d $(DIR) instrument -w /tmp/work --keep-work -o clientgood.instr dynagrok/examples/shapes/client

run: install clientgood
