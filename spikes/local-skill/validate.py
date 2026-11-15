"""Use the upstream format validator plus repository resource checks."""
import os
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
SKILL = ROOT / 'examples/skills/renodash-local-advisory'
TOOL = 'git+https://github.com/agentskills/agentskills.git@83021e2be457a9b6679148719c1f6074f0ad975e#subdirectory=skills-ref'
subprocess.run(['uv', 'tool', 'run', '--cache-dir', '/tmp/renodash-skills-uv',
                '--from', TOOL, 'skills-ref', 'validate', str(SKILL)],
               env=dict(os.environ, UV_TOOL_DIR='/tmp/renodash-skills-tools'), check=True)
text = (SKILL / 'SKILL.md').read_text()
for target in re.findall(r'\[[^\]]+\]\(([^)]+)\)', text):
    if '://' not in target and not target.startswith('#'):
        assert (SKILL / target.split('#')[0]).is_file(), target
bundle = Path(sys.argv[1]).resolve()
# Commands referenced by this Skill are resources supplied by the prepared bundle.
for resource in ('bin/advisory', 'contracts.py', 'provenance.json'):
    assert (bundle / 'runtime' / resource).is_file(), resource
subprocess.run(['python3', str(bundle / 'runtime/contracts.py'),
                str(bundle / 'runtime/bin/advisory')], check=True)
print('Upstream format, linked files, command resources, adapter contracts: PASS')
