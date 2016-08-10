#!/usr/bin/env python2

import sys
import random
random.seed()

import optutils

from random_tree import Remote, Map

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

    def _execute(self):
        remote = Remote(self.binary, '/tmp/dynamic-callgraph.dot')
        table = Map(remote)
        success = True
        for i, op in enumerate(self.case):
            #print 'executing', op
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

    def hash(self):
        if self._hash is None:
            self._hash = hash(tuple(tuple(line) for line in self.case))
        return self._hash

    def mutations(self):
        self.trim()
        prefix = [
            (0, i)
            for i in xrange(1, len(self.case)-2)
        ]
        blocks = [
            (i, j)
            for i in xrange(1, len(self.case)-1)
            for j in xrange(i+1, min(i+min(max(2, int(.02*len(self.case))), 10), len(self.case)))
        ]
        return prefix + blocks

    def kids(self):
        cases = list()
        for m in self.mutations():
            m = self.remove_block(*m)
            _, ok = m.execute()
            if not ok:
                m.trim()
                cases.append(m)
        return cases

    def trim(self):
        i, ok = self.execute()
        self.case = self.case[:i+1]

    def remove(self, i):
        return Testcase(self.binary, self.case[:i] + self.case[i+1:])

    def remove_prefix(self, i):
        return Testcase(self.binary, self.case[i+1:])

    def remove_block(self, i, j):
        return Testcase(self.binary, self.case[:i] + self.case[j:])

def minimize(testcase):
    _, ok = testcase.execute()
    if ok:
        raise Exception, "non failing test case"
    testcase.trim()
    stack = list()
    stack.append(testcase)
    min_case = None
    seen = set()
    while len(stack) > 0:
        c = stack.pop()
        if c.hash() in seen:
            continue
        seen.add(c.hash())
        print "cur case", len(c.case)
        if min_case is None or len(c.case) < len(min_case.case):
            min_case = c
            with open("/tmp/min.case", "w") as f:
                for line in c.case:
                    print >>f, ' '.join(line)
        for k in c.kids():
            if k.hash() in seen:
                continue
            stack.append(k)
    return min_case

def walk(testcase):
    _, ok = testcase.execute()
    if ok:
        raise Exception, "non failing test case"
    testcase.trim()
    p = testcase
    c = testcase
    while c is not None:
        print 'cur', len(c.case)
        p = c
        kid = None
        seen = set()
        mutations = c.mutations()
        while kid is None and len(mutations) > 0:
            i = random.randint(0, len(mutations) - 1)
            kid = mutations.pop(i)
            kid = c.remove_block(*kid)
            _, ok = kid.execute()
            if ok:
                kid = None
            else:
                kid.trim()
                break
        c = kid
    print mutations
    print 'found', len(p.case)
    with open("/tmp/min-%d.case" % len(p.case), "w") as f:
        for line in p.case:
            print >>f, ' '.join(line)
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
        case = [
            line.strip().split(' ')
            for line in f
            if line.strip()
        ]
    t = Testcase(args[0], case)
    #print minimize(t)
    c = min(
        sample(t, 1),
        key=lambda c: len(c.case),
    )
    print "min case", len(c.case)
    for line in c.case:
        print ' '.join(line)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
