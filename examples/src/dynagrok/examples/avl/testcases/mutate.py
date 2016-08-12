#!/usr/bin/env python2

import sys
import random
random.seed()

import optutils

from map_tester import Remote, Map

class Testcase(object):

    def __init__(self, binary, case):
        self.binary = binary
        self.case = case
        self.executed = False
        self.ok = None
        self.i = None
        self._hash = None

    def execute(self):
        if self.executed:
            return self.i, self.ok
        self.i, self.ok = self._execute()
        self.executed = True
        return self.i, self.ok

    def ops(self):
        ops = list()
        for line in self.case.split('\n'):
            if line == 'verify': continue
            if line == 'print': continue
            if line == 'serialize': continue
            ops.append(line.split(' '))
        return ops

    def valid(self):
        return all(Map.valid(op) for op in self.ops())

    def _execute(self):
        remote = Remote(self.binary, '/tmp/dynamic-callgraph.dot')
        table = Map(remote)
        success = True
        if not self.valid():
            raise Exception("Not a valid case")
        i = 0
        for i, op in enumerate(self.ops()):
            #print 'executing', i, op
            ok = table.do(op)
            if not ok:
                success = False
                break
            #print 'executed', op, ok
            ok = table.verify()
            if not ok:
                success = False
                break
            #print 'executed', 'verify', ok
        remote.close()
        return i, success

    def static_case(self):
        return self.case

    def __eq__(self, b):
        return self.static_case == b.static_case

    def __hash__(self):
        return self.hash()

    def hash(self):
        if self._hash is None:
            self._hash = hash(self.static_case())
        return self._hash

    def mutations(self):
        prefix = [
            (0, i)
            for i in xrange(1, len(self.case)-1)
        ]
        blocks = [
            (i, j)
            for i in xrange(1, len(self.case))
            for j in xrange(i+1, min(i+min(max(15, int(.1*len(self.case))), 100), len(self.case)+1))
        ]
        return prefix + blocks

    def trim(self):
        i, ok = self.execute()
        self.case = self.case[:i+1]
        self.hash()

    def remove_block(self, i, j):
        return Testcase(self.binary, self.case[:i] + self.case[j:])

def walk(testcase):
    _, ok = testcase.execute()
    if ok:
        raise Exception, "non failing test case"
    p = testcase
    c = testcase
    mut = None
    tries = 0
    while c is not None:
        print 'cur', len(c.case), mut, tries
        with open("/tmp/cur.case", "w") as f:
            print >>f, c.case
        p = c
        kid = None
        mutations = c.mutations()
        tries = 0
        while kid is None and len(mutations) > 0:
            tries += 1
            i = random.randint(0, len(mutations) - 1)
            mut = mutations.pop(i)
            kid = c.remove_block(*mut)
            if kid.valid():
                #print 'valid', kid.case
                _, ok = kid.execute()
                if ok:
                    kid = None
                else:
                    break
            else:
                kid = None
        c = kid
    print 'found', len(p.case), mut, tries #, p.mutations()
    with open("/tmp/min-%d.case" % len(p.case), "w") as f:
        print >>f, p.case
    return p

def sample(testcase, n):
    return [
        walk(testcase)
        for x in xrange(n)
    ]

@optutils.main(
'mutate.py -t <testcase> <program>',
'''
Options
-h, --help                              show this message
-t, --testcase=<path>                   test case to mutate
''',
'ht:',
['help', 'testcase='],
)
def main(argv, util, parser):
    testcase = None
    opts, args = parser(argv)
    for opt, arg in opts:
        if opt in ('-h', '--help',):
            util.usage()
        elif opt in ('-t', '--testcase',):
            testcase = util.assert_file_exists(arg)
        else:
            util.log("unknown option %s" % (opt))
            util.usage(1)

    if testcase is None:
        util.log("must supply a testcase")
        util.usage(1)

    if len(args) != 1:
        util.log("must supply a path the program under test")
        util.log("got %s" % args)
        util.usage(1)

    with open(testcase) as f:
        case = f.read().strip()
    t = Testcase(args[0], case)
    #t.trim()
    #print minimize(t)
    c = min(
        sample(t, 1),
        key=lambda c: len(c.case),
    )
    print "min case", len(c.case)
    print c.case


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
