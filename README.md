# Dynagrok
Dynagrok is a dynamic analysis tool for Go. It generates control flow graphs,
call graphs and control depence graphs. It even has a random code mutator.
It also performs a number of analyses, such as:
* Code clone detection
* Statistical fault localization
* Test case pruning

## Installation
The following process chooses ~/dev/go-research for dynagrok's GOROOT,
~/dev/dynagrok for dynagrok's GOPATH. You can set these to different locations,
but make sure you're consistent about it.


### Step 0: Preliminary
#### 0.1: Install go
Make sure you have a working go installation, with proper configuration. The
[official website](https://golang.org) can help with this.

#### 0.2: Install dep
Dep is the unofficial dependency management tool of Go. It is required to set up
dynagrok for the first time.
Run the following command:
```bash
go get -u github.com/golang/dep/cmd/dep
```

### Step 1: Compile a compiler
For compiling instrumented binaries, Dynagrok uses an augmented version of the
Go compiler. Rather than messing with the Go compiler you use every day,
you'll want to have a separate one for use with Dynagrok.

#### 1.1: Install go1.4
Go compilers compile themselves, but they need help from a bootstrap compiler.
Go 1.4 is our bootstrap compiler.
We'll install this to ~/go1.4, but you can put it wherever you like, as long as
you set $GO_BOOTSTRAP to that directory.

Install the right go1.4 for your platform. Binaries can be found [here](https://golang.org/dl/#go1.4).
You may follow the instructions [here](https://golang.org/doc/install) but do
not set your GOROOT or GOPATH to this.

#### 1.2: Clone the compiler and checkout the right version
We'll clone this to ~/dev
```bash
cd ~/dev
git clone https://go.googlesource.com/go go-research
cd go-research
git checkout release-branch.go1.8
```
#### 1.3: Build from source
``` bash
cd ~/dev/go-research/src
./all.bash
```
### Step 2: Create an isolated GOPATH
We'll create this at ~/dev/dynagrok,
and it will be the root of the GOPATH for dynagrok, but the project itself will
live in ~/dev/dynagrok/src/github.com/timtadh/dynagrok
```bash
mkdir -p ~/dev/dynagrok/{src,bin,lib}
mkdir -p ~/dev/dynagrok/src/github.com/timtadh/
```

### Step 3: Install dynagrok
``` bash
cd ~/dev/dynagrok/src/github.com/timtadh
git clone http://github.com/timtadh/dynagrok
cd ~/dev/dynagrok/src/github.com/timtadh/dynagrok
git submodule init # initialize submodules
git submodule update # install submodules (requires github to have your ssh key)
```
The `.activate` script sets the environment to their proper values for a dynagrok
session. **It must be sourced before every session.** The last line of the file
is for building a sub-utility and ought to be removed.

```bash
vim .activate # Edit `.activate` to remove the last line
source .activate
```
Then,
```bash
dep ensure # install remaining dependencies
```

### Step 4: Test your installation
Build dynagrok
```bash
go install github.com/timtadh/dynagrok
```
Compile an example program:
```bash
dynagrok -r ~/dev/go-research -d ~/dev/dynagrok/src/github.com/timtadh/dynagrok
-g ~/dev/dynagrok/src/github.com/timtadh/dynagrok/examples objectstate
--keep-work -w /tmp/work dynagrok/examples/linkedlist
```

## Usage
At the start of each dynagrok session, make sure to run
```bash
source .activate
```
`dynagrok --help` should also be helpful for viewing usage information.

## Under the hood

