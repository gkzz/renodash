"""Run only inside the disposable offline container. No real GitHub writes."""
import errno
import json
import os
from pathlib import Path
import socket
import subprocess

assert os.getuid() != 0
status = Path('/proc/self/status').read_text()
assert 'CapEff:\t0000000000000000' in status
assert 'NoNewPrivs:\t1' in status
for path in ('/spike/provenance.json', '/etc/renodash-probe'):
    try:
        with open(path, 'ab') as output:
            output.write(b'forbidden')
    except OSError as error:
        assert error.errno in (errno.EROFS, errno.EACCES), error
    else:
        raise AssertionError('write permitted: ' + path)
for path in ('/var/run/docker.sock', '/host-canary', '/root/.ssh', '/root/.config/gh'):
    assert not os.path.exists(path), path
assert not any(key in os.environ for key in ('GH_TOKEN', 'GITHUB_TOKEN', 'SSH_AUTH_SOCK', 'AWS_SECRET_ACCESS_KEY'))
# Numeric address avoids mistaking a DNS failure for a network barrier.
try:
    socket.create_connection(('1.1.1.1', 443), timeout=2)
except OSError:
    pass
else:
    raise AssertionError('external connection succeeded')
# All these write paths require the same blocked transport. No real operation is sent.
for operation in ('git push', 'gh pr merge', 'gh pr comment', 'gh issue comment', 'repository settings API', 'artifact upload API'):
    try:
        socket.create_connection(('140.82.112.6', 443), timeout=2)
    except OSError:
        print(operation + ': transport denied')
    else:
        raise AssertionError('GitHub transport available')
subprocess.run(['python3', '/spike/contracts.py', '/spike/bin/advisory'], check=True)
print(json.dumps({'isolation': 'passed', 'model_eval': 'not run'}))
