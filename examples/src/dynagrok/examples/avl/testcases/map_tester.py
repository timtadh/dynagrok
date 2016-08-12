#!/usr/bin/env python2

import os
import sys
import subprocess
import random

import optutils


random.seed()

def isint(a):
    try:
        int(a)
        return True
    except ValueError, e:
        return False

class Remote(object):

    def __init__(self, path, dgpath):
        env = os.environ
        env['DGPROF'] = dgpath
        self.p = subprocess.Popen([path],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                env=env,
        )
        self.input = list()
        self.output = list()

    def render(self, op):
        op = [str(arg) if not isint(arg) else str(int(arg))
              for i, arg in enumerate(op)]
        return ' '.join(op)

    def get_output(self):
        line = self.p.stdout.readline().strip()
        self.output.append(line)
        #print '>', line
        return line

    def send_input(self, inp):
        #print '$', inp
        self.p.stdin.write(inp)
        self.p.stdin.write('\n')
        self.input.append(inp)

    def ex(self, op):
        self.send_input(self.render(op))
        line = self.get_output().split()
        if not line:
            return False, list()
        if line[0] != "ex":
            return False, list()
        return True, line[1:]

    def close(self):
        self.p.communicate()[0]


class Map(object):

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
            raise Exception('exec failed of verify')
            return False
        if res[1] != 'true':
            return False
        return True

    def string(self):
        ok, res = self.remote.ex(['print'])
        if not ok:
            print 'exec failed of print'
            return False
        return ' '.join(res[1:])

    @staticmethod
    def valid(op):
        if len(op) <= 0:
            return False
        if op[0] == 'put' and len(op) == 3:
            return isint(op[1]) and isint(op[2])
        elif op[0] == 'rm' and len(op) == 2:
            return isint(op[1])
        elif op[0] == 'has' and len(op) == 2:
            return isint(op[1])
        elif op[0] == 'get' and len(op) == 2:
            return isint(op[1])
        else:
            return False

    def do(self, op):
        exp = list()
        if op[0] == 'put':
            self.model[int(op[1])] = int(op[2])
            exp = ['put', op[1], op[2]]
        elif op[0] == 'rm':
            self.model.pop(int(op[1]), None)
            exp = ['rm', op[1]]
        elif op[0] == 'has':
            exp = ['has', str(int(op[1]) in self.model).lower()]
        elif op[0] == 'get':
            exp = ['get', self.model.get(int(op[1]), 0), str(int(op[1]) in self.model).lower()]
        else:
            raise Exception("unexpected op %s" % str(op))
        ok, res = self.remote.ex(op)
        if not ok:
            print 'exec failed of', op
            return False
        exp = self.remote.render(exp)
        if exp != self.remote.render(res):
            print 'failed op', self.remote.render(op)
            print 'expected', exp
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


@optutils.main(
'random_tree.py -o <output-dir> <program>',
'''
Options
-h, --help                              show this message
-o, --output-dir=<path>                 output directory
''',
'ho:',
['help', 'output='],
)
def main(argv, util, parser):
    output = None
    opts, args = parser(argv)
    for opt, arg in opts:
        if opt in ('-h', '--help',):
            util.usage()
        elif opt in ('-o', '--output',):
            output = util.assert_dir_exists(arg)
        else:
            util.log("unknown option %s" % (opt))
            util.usage(1)


    if output is None:
        util.log("must supply an output directory")
        util.usage(1)

    if len(args) != 1:
        util.log("you  must supply a path to the program")
        util.usage(1)

    program = args[0]
    t = Map(Remote(program, os.path.join(output, 'dynamic-callgraph.dot')))

    fail = False
    for x in xrange(random.randint(10, 1000)):
        ok = t.op()
        if not ok:
            fail = True
            break
    if not t.verify():
        fail = True
    print t.string()
    t.remote.close()

    with open(os.path.join(output, 'input'), 'w') as f:
        for line in t.remote.input:
            print >>f, line
    with open(os.path.join(output, 'output'), 'w') as f:
        for line in t.remote.output:
            print >>f, line
    with open(os.path.join(output, 'result'), 'w') as result:
        if fail:
            print >>result, "FAIL"
            return 1
        else:
            print >>result, "OK"
            return 0

if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
