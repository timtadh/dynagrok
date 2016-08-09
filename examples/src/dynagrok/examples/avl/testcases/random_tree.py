#!/usr/bin/env python2

import os
import sys
import subprocess
import random

random.seed()


class Remote(object):

    def __init__(self, path):
        self.p = subprocess.Popen([path],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
        )

    def render(self, op):
        return ' '.join([str(x) for x in op])

    def get_line(self):
        return self.p.stdout.readline().strip()

    def ex(self, op):
        op = self.render(op)
        print '$', op
        self.p.stdin.write(op)
        self.p.stdin.write('\n')
        line = self.get_line().split()
        #line = ['ex', 'wacky', 'true']
        print '>', ' '.join(line)
        if line[0] != "ex":
            return False, list()
        return True, line[1:]

    def close(self):
        print self.p.communicate()[0]


class Tree(object):

    def __init__(self, remote):
        self.model = dict()
        self.remote = remote

    def op(self):
        ops = [self.put, self.rm, self.has, self.get]
        op = random.choice(ops)
        return self.do(op())

    def verify(self):
        ok, res = self.remote.ex(['verify'])
        if not ok:
            print 'exec failed of verify'
            return False
        print res
        if res[1] != 'true':
            return False
        return True

    def string(self):
        ok, res = self.remote.ex(['print'])
        if not ok:
            print 'exec failed of print'
            return False
        return ' '.join(res[1:])

    def do(self, op):
        exp = list()
        if op[0] == 'put':
            self.model[op[1]] = op[2]
            exp = ['put', op[1], op[2]]
        elif op[0] == 'rm':
            self.model.pop(op[1], None)
            exp = ['rm', op[1]]
        elif op[0] == 'has':
            exp = ['has', str(op[1] in self.model).lower()]
        elif op[0] == 'get':
            exp = ['get', self.model.get(op[1], 0), str(op[1] in self.model).lower()]
        else:
            raise Exception("unexpected op %s" % str(op))
        ok, res = self.remote.ex(op)
        if not ok:
            print 'exec failed of', op
            return False
        if self.remote.render(exp) != self.remote.render(res):
            print 'expected', self.remote.render(exp)
            print 'got', self.remote.render(res)
            return False
        return True

    def put(self):
        k = random.randint(0, 10000)
        v = random.randint(0, 10000)
        return ['put', k, v]

    def key(self):
        if len(self.model) > 0 and random.random() > .5:
            k = random.choice(self.model.keys())
        else:
            k = random.randint(0, 10000)
        return k

    def rm(self):
        return ['rm', self.key()]

    def has(self):
        return ['has', self.key()]

    def get(self):
        return ['get', self.key()]


def main(argv):
    if len(argv) != 1:
        print "expected path of avl program"
        sys.exit(1)
    print argv
    avl_path = argv[0]
    t = Tree(Remote(avl_path))
    for x in xrange(random.randint(10, 1000)):
        ok = t.op()
        if not ok:
            print "FAIL"
            sys.exit(1)
    ok = t.verify()
    if not ok:
        print 'FAIL'
        sys.exit(1)
    print t.string()
    t.remote.close()
    print 'OK'
    sys.exit(0)

if __name__ == "__main__":
    main(sys.argv[1:])
