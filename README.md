# Dynagrok
Dynagrok is an instrumenting and profiling tool for go code. On the object-state
branch, it profiles object-state.

## Installation
If you've gotten this far, we will assume you've already executed
`git checkout object-state`
After that, from the project's parent directory:
```bash
git submodule init
git submodule update
source .activate
make install
```

## Usage
For your convenience, an example usage is included as a make target.
```bash
make example
```
Executes the equivalent of the following command:
```bash
dynagrok -g $(pwd)/examples -d $(pwd) instrument -w /tmp/work --keep-work -o example.instr dynagrok/examples/shapes/client
```
