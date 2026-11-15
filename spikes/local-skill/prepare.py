"""Export existing adapters without changing branches; build an offline bundle."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
REVISIONS = {
    "published-advisory": "2654fc92c0bb4c900b1d23dd5ae6af3caf4b6870",
    "govulncheck-discovery": "c27bcd21baf8196d2eabeddfa28427763048f415",
}


def prepare(destination):
    destination.mkdir()  # Never overwrite an existing bundle.
    for name, revision in REVISIONS.items():
        prefix = f"adapters/{name}/"
        files = subprocess.check_output(
            ["git", "ls-tree", "-r", "--name-only", revision, "--", prefix], cwd=ROOT,
            text=True).splitlines()
        if not files:
            raise RuntimeError(f"Missing adapter at {revision}; fetch the documented spike branch")
        for filename in files:
            target = destination / filename
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(subprocess.check_output(
                ["git", "show", f"{revision}:{filename}"], cwd=ROOT))
    (destination / "runtime/bin").mkdir(parents=True)
    env = dict(os.environ, GOCACHE="/tmp/renodash-spike-go-cache", CGO_ENABLED="0", GOOS="linux", GOARCH="amd64")
    subprocess.run(["go", "build", "-trimpath", "-buildvcs=false", "-o", str(destination / "runtime/bin/advisory"), "."],
                   cwd=destination / "adapters/published-advisory", env=env, check=True)
    for filename in ("contracts.py", "probe.py"):
        shutil.copyfile(Path(__file__).parent / filename, destination / "runtime" / filename)
    shutil.copytree(ROOT / "examples/skills/renodash-local-advisory", destination / "runtime/renodash-local-advisory")
    (destination / "runtime/provenance.json").write_text(json.dumps(REVISIONS, indent=2) + "\n")
    shutil.copyfile(ROOT / "LICENSE", destination / "runtime/LICENSE")
    # Real saved advisory captures remain outside the runtime mount.
    print(destination)


if __name__ == "__main__":
    prepare(Path(sys.argv[1]).resolve())
