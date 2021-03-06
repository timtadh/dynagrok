#!/usr/bin/env python2

import os
import sys
import subprocess
import random
import time
import resource
import signal
import fcntl
import errno

import optutils


random.seed()

def isint(a):
    try:
        int(a)
        return True
    except ValueError, e:
        return False

def _eintr_retry_call(func, *args):
    while True:
        try: 
            return func(*args)
        except (OSError, IOError) as e:
            if e.errno == errno.EINTR:
                time.sleep(.00001)
                continue
            if e.errno == errno.EAGAIN:
                return ''
            raise

class Remote(object):

    MEM_LIMIT = int(50 * 10**7) ### 500 MB
    TIME_LIMIT = 2 ### 2 seconds

    def __init__(self, path, dgpath):
        env = os.environ
        env['DGPROF'] = dgpath
        self.p = subprocess.Popen(
                [path],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                env=env,
                #bufsize=4096,
        )
        fd = self.p.stdout.fileno()
        fl = fcntl.fcntl(fd, fcntl.F_GETFL)
        fcntl.fcntl(fd, fcntl.F_SETFL, fl | os.O_NONBLOCK)
        self.input = list()
        self.output = list()
        self.buf = ''

    def kill(self):
        self.p.kill()
        self.p.wait()

    def getmem(self):
        ### memory usage as measure in pages
        path = '/proc/%s/statm' % str(self.p.pid)
        with open(path, 'r') as f:
            statm = f.read()
        total = int(statm.split(' ')[1]) ## resident size
        return total * resource.getpagesize()

    def over_time(self, start_time):
        mem = self.getmem()
        if mem > self.MEM_LIMIT:
            print "You used too much memory, killing"
            return True
        if time.time() - start_time > self.TIME_LIMIT:
            print "You used too much time, killing"
            return True
        return False

    def render(self, op):
        op = [str(arg) if not isint(arg) else str(int(arg))
              for i, arg in enumerate(op)]
        return ' '.join(op)

    def get_output(self):
        if self.p.returncode is not None:
            raise Exception("read on closed process")
        start_time = time.time()
        while '\n' not in self.buf:
            if self.over_time(start_time):
                self.kill()
                return ''
            text = _eintr_retry_call(self.p.stdout.read, 1)
            if text != '':
                self.buf += text
            else:
                time.sleep(.0001)
        line, self.buf = self.buf.split('\n', 1)
        line = line.strip()
        self.output.append(line)
        #print '>', line
        return line

    def send_input(self, inp):
        if self.p.returncode is not None:
            raise Exception("send on closed process")
        #print '$', inp
        self.p.stdin.write(inp)
        self.p.stdin.write('\n')
        self.input.append(inp)

    def ex(self, op):
        if self.p.returncode is not None:
            return False, list()
        self.send_input(self.render(op))
        line = self.get_output().split()
        if not line:
            return False, list()
        if line[0] != "ex":
            return False, list()
        return True, line[1:]

    def close(self):
        if self.p.returncode is None:
            self.p.communicate()


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
            print 'exec failed of verify'
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
    t = Map(Remote(program, os.path.join(output, 'dgprofile')))

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
    with open(os.path.join(output, 'result'), 'w') as f:
        if fail:
            print >>f, "FAIL"
            return 1
        else:
            print >>f, "OK"
            return 0

if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
