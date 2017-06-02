# Dynagrok
Dynagrok is a dynamic analysis tool for Go. It generates control flow graphs,
call graphs and control depence graphs. It even has a random code mutator.
It also performs a number of analyses, such as:
* Code clone detection
* Statistical fault localization
* Test case pruning

## Installation
### Step 1: Compile a compiler
Dynagrok uses an augmented version of the Go compiler. Rather than messing with the
Go compiler you use every day, you'll want to have a separate one for use with
Dynagrok.

#### Install go1.4 
We'll install this to ~/go1.4 (but you can put it wherever you like, as long as
you set $GO_BOOTSTRAP to that directory)
Install the right go1.4 for your platform. Binaries can be found [here](https://golang.org/dl/#go1.4).
You may follow the instructions [here](https://golang.org/doc/install) but do
not set your GOROOT or GOPATH to this.
#### Clone the compiler and checkout the right version
We'll clone this to ~/dev
```bash
cd ~/dev
git clone https://go.googlesource.com/go go-research
cd go-reserarch
git checkout release-branch.go1.8
```
#### Build from source
From ~/dev/go-research:
``` bash
./all.bash
```
### Step 2: Create an isolated GOPATH
We'll create this at ~/dev/dynagrok
```bash
mkdir -p ~/dev/dynagrok/{src,bin,lib}
mkdir -p ~/dev/dynagrok/src/github.com/timtadh/
```

### Step 3: Install dynagrok
``` bash
cd ~/dev/dynagrok/src/github.com/timtadh
git clone http://github.com/timtadh/dynagrok
dep ensure # install dependencies
git submodule init # install submodules
git submodule update
```
Edit `.activate` and remove the last line, and then
```bash
source .activate
```

### Step 4: Test your installation
Build dynagrok with `make`
Compile an example program with `make prog` (the default is a linked list
program)

## Usage

## Under the hood

