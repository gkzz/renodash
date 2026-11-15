"""Launch the actual Dev Container CLI, audit effective settings, then probe."""
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

HERE = Path(__file__).resolve().parent
bundle = Path(sys.argv[1]).resolve()
assert (bundle / 'runtime/bin/advisory').is_file()
with tempfile.TemporaryDirectory(prefix='renodash-devcontainer-') as home:
    canary = Path(home) / 'host-canary'
    canary.write_text('synthetic host secret; never mount')
    env = {'PATH': os.environ['PATH'], 'HOME': home, 'RENODASH_SPIKE_BUNDLE': str(bundle)}
    result = subprocess.run([
        'devcontainer', 'up', '--workspace-folder', str(bundle),
        '--config', str(HERE / '.devcontainer/devcontainer.json'),
        '--mount-workspace-git-root', 'false', '--skip-post-create',
        '--container-session-data-folder', '/tmp/devcontainer-session',
    ], env=env, text=True, capture_output=True)
    if result.returncode:
        failed = json.loads(result.stdout.splitlines()[-1])
        if failed.get('containerId'):
            subprocess.run(['docker', 'rm', '-f', failed['containerId']], check=True, stdout=subprocess.DEVNULL)
        sys.stderr.write(result.stderr)
        sys.stderr.write(result.stdout)
        raise SystemExit(result.returncode)
    cid = json.loads(result.stdout.splitlines()[-1])['containerId']
    try:
        info = json.loads(subprocess.check_output(['docker', 'inspect', cid]))[0]
        config, host = info['Config'], info['HostConfig']
        assert config['User'] == 'spike'
        assert host['NetworkMode'] == 'none' and host['ReadonlyRootfs']
        assert host['CapDrop'] == ['ALL']
        assert 'no-new-privileges:true' in host['SecurityOpt']
        mounts = info['Mounts']
        assert len(mounts) == 1 and mounts[0]['Destination'] == '/spike' and not mounts[0]['RW'], mounts
        assert Path(mounts[0]['Source']).resolve() == bundle / 'runtime'
        assert not any(s.split('=')[0] in ('GH_TOKEN', 'GITHUB_TOKEN', 'SSH_AUTH_SOCK', 'AWS_SECRET_ACCESS_KEY') for s in config['Env'])
        subprocess.run(['docker', 'exec', cid, 'python3', '/spike/probe.py'], check=True)
        # A random, existing host path is not visible from the container.
        subprocess.run(['docker', 'exec', cid, 'python3', '-c',
                        'import os,sys; assert not os.path.exists(sys.argv[1])', str(canary)], check=True)
        print('Effective Dev Container configuration and non-shared host canary: PASS')
    finally:
        subprocess.run(['docker', 'rm', '-f', cid], check=True, stdout=subprocess.DEVNULL)
