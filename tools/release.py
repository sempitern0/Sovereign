#!/usr/bin/env python3
"""Build reproducible, verifiable suitectl release artifacts (ADR-0008).

For every target this produces an archive (zip on Windows, tar.gz on Linux),
an SPDX 2.3 SBOM and finally SHA256SUMS. Output depends only on the source
tree, the Go toolchain, --version, the commit and SOURCE_DATE_EPOCH: building
twice must give byte-identical files. Python is used at development and
release time only; end users never need it.
"""
from __future__ import annotations

import argparse
import gzip
import hashlib
import io
import json
import os
import re
import subprocess
import sys
import tarfile
import tempfile
import zipfile
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MODULE = "github.com/sempitern0/Sovereign"
REPO_URL = "https://github.com/sempitern0/Sovereign"
TARGETS = ("linux/amd64", "linux/arm64", "windows/amd64", "windows/arm64")
DOCS = ("README.md", "LICENSE", "SECURITY.md", "CHANGELOG.md")
SEMVER = re.compile(r"^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z.-]+)?$")
ZIP_EPOCH_MIN = 315532800  # 1980-01-01, the earliest zip timestamp


def run(cmd: list[str], env: dict[str, str] | None = None) -> str:
    return subprocess.run(cmd, cwd=ROOT, env=env, check=True, capture_output=True, text=True).stdout.strip()


def artifact_base(version: str, target: str) -> str:
    goos, goarch = target.split("/")
    return f"suitectl_{version}_{goos}_{goarch}"


def build_binary(version: str, commit: str, target: str, out: Path) -> Path:
    goos, goarch = target.split("/")
    binary = out / ("suitectl.exe" if goos == "windows" else "suitectl")
    env = {**os.environ, "CGO_ENABLED": "0", "GOOS": goos, "GOARCH": goarch, "GOFLAGS": "-mod=vendor"}
    ldflags = f"-s -w -buildid= -X main.version={version} -X main.commit={commit}"
    run(["go", "build", "-trimpath", "-buildvcs=false", "-ldflags", ldflags, "-o", str(binary), "./cmd/suitectl"], env)
    return binary


def archive_members(binary: Path, base: str) -> list[tuple[str, bytes, int]]:
    """Archive entries as (path, content, mode), sorted by path."""
    members = [(f"{base}/{binary.name}", binary.read_bytes(), 0o755)]
    members += [(f"{base}/{doc}", (ROOT / doc).read_bytes(), 0o644) for doc in DOCS]
    return sorted(members)


def write_tar_gz(path: Path, members: list[tuple[str, bytes, int]], epoch: int) -> None:
    raw = io.BytesIO()
    with tarfile.open(fileobj=raw, mode="w", format=tarfile.PAX_FORMAT) as tar:
        for name, data, mode in members:
            info = tarfile.TarInfo(name)
            info.size, info.mode, info.mtime = len(data), mode, epoch
            info.uid = info.gid = 0
            info.uname = info.gname = ""
            tar.addfile(info, io.BytesIO(data))
    with open(path, "wb") as handle, gzip.GzipFile(filename="", mode="wb", fileobj=handle, mtime=epoch, compresslevel=9) as gz:
        gz.write(raw.getvalue())


def write_zip(path: Path, members: list[tuple[str, bytes, int]], epoch: int) -> None:
    stamp = datetime.fromtimestamp(max(epoch, ZIP_EPOCH_MIN), tz=timezone.utc).timetuple()[:6]
    with zipfile.ZipFile(path, "w") as archive:
        for name, data, mode in members:
            info = zipfile.ZipInfo(name, date_time=stamp)
            info.compress_type = zipfile.ZIP_DEFLATED
            info.create_system = 3  # Unix, so the mode bits below are honoured
            info.external_attr = (0o100000 | mode) << 16
            archive.writestr(info, data, compresslevel=9)


def parse_go_version_m(text: str) -> dict:
    """Extract Go version, main module and dependencies from `go version -m`."""
    info: dict = {"go": "", "deps": []}
    for index, line in enumerate(text.splitlines()):
        if index == 0:
            info["go"] = line.rsplit(" ", 1)[-1].strip()
            continue
        fields = line.strip().split("\t")
        if fields[0] == "dep" and len(fields) >= 3:
            info["deps"].append({"path": fields[1], "version": fields[2]})
    return info


def spdx_id(text: str) -> str:
    return "SPDXRef-Package-" + re.sub(r"[^A-Za-z0-9.-]", "-", text)


def build_sbom(base: str, version: str, binary_name: str, binary_sha256: str, build: dict, epoch: int) -> dict:
    created = datetime.fromtimestamp(epoch, tz=timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    main_id = "SPDXRef-Package-suitectl"
    packages = [{
        "SPDXID": main_id,
        "name": "suitectl",
        "versionInfo": version,
        "downloadLocation": f"{REPO_URL}/releases/download/v{version}/{base}",
        "filesAnalyzed": False,
        "licenseConcluded": "MIT",
        "licenseDeclared": "MIT",
        "copyrightText": "NOASSERTION",
        "packageFileName": binary_name,
        "checksums": [{"algorithm": "SHA256", "checksumValue": binary_sha256}],
        "externalRefs": [{"referenceCategory": "PACKAGE-MANAGER", "referenceType": "purl",
                          "referenceLocator": f"pkg:golang/{MODULE.lower()}@v{version}"}],
    }, {
        "SPDXID": spdx_id("golang-stdlib"),
        "name": "golang/stdlib",
        "versionInfo": build["go"],
        "downloadLocation": "https://go.dev/dl/",
        "filesAnalyzed": False,
        "licenseConcluded": "NOASSERTION",
        "licenseDeclared": "BSD-3-Clause",
        "copyrightText": "NOASSERTION",
        "externalRefs": [{"referenceCategory": "PACKAGE-MANAGER", "referenceType": "purl",
                          "referenceLocator": f"pkg:golang/stdlib@{build['go']}"}],
    }]
    for dep in sorted(build["deps"], key=lambda d: d["path"]):
        packages.append({
            "SPDXID": spdx_id(dep["path"]),
            "name": dep["path"],
            "versionInfo": dep["version"],
            "downloadLocation": "NOASSERTION",
            "filesAnalyzed": False,
            "licenseConcluded": "NOASSERTION",
            "licenseDeclared": "NOASSERTION",
            "copyrightText": "NOASSERTION",
            "externalRefs": [{"referenceCategory": "PACKAGE-MANAGER", "referenceType": "purl",
                              "referenceLocator": f"pkg:golang/{dep['path']}@{dep['version']}"}],
        })
    relationships = [{"spdxElementId": "SPDXRef-DOCUMENT", "relationshipType": "DESCRIBES", "relatedSpdxElement": main_id}]
    relationships += [{"spdxElementId": main_id, "relationshipType": "DEPENDS_ON", "relatedSpdxElement": p["SPDXID"]}
                      for p in packages[1:]]
    return {
        "spdxVersion": "SPDX-2.3",
        "dataLicense": "CC0-1.0",
        "SPDXID": "SPDXRef-DOCUMENT",
        "name": base,
        "documentNamespace": f"{REPO_URL}/releases/download/v{version}/{base}.spdx.json",
        "creationInfo": {"created": created, "creators": ["Tool: sovereign/tools/release.py"]},
        "packages": packages,
        "relationships": relationships,
    }


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def release(version: str, commit: str, epoch: int, out: Path, targets: tuple[str, ...] = TARGETS) -> list[Path]:
    out.mkdir(parents=True, exist_ok=True)
    produced: list[Path] = []
    for target in targets:
        base = artifact_base(version, target)
        with tempfile.TemporaryDirectory() as work:
            binary = build_binary(version, commit, target, Path(work))
            members = archive_members(binary, base)
            archive = out / (base + (".zip" if target.startswith("windows/") else ".tar.gz"))
            (write_zip if archive.suffix == ".zip" else write_tar_gz)(archive, members, epoch)
            build = parse_go_version_m(run(["go", "version", "-m", str(binary)]))
            sbom = build_sbom(base, version, binary.name, sha256(binary), build, epoch)
            sbom_path = out / f"{base}.spdx.json"
            sbom_path.write_text(json.dumps(sbom, indent=2, sort_keys=True) + "\n", encoding="utf-8")
            produced += [archive, sbom_path]
    sums = out / "SHA256SUMS"
    sums.write_text("".join(f"{sha256(p)}  {p.name}\n" for p in sorted(produced)), encoding="utf-8")
    return produced + [sums]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--version", required=True, help="semantic version without the leading v, e.g. 0.3.0")
    parser.add_argument("--out", default="dist", help="output directory (default: dist)")
    parser.add_argument("--commit", help="git commit (default: HEAD)")
    parser.add_argument("--source-date-epoch", type=int, help="timestamp for archives (default: SOURCE_DATE_EPOCH or HEAD commit time)")
    parser.add_argument("--target", action="append", choices=TARGETS, help="restrict to one or more targets")
    args = parser.parse_args()
    if not SEMVER.match(args.version):
        parser.error("--version must be a semantic version without a leading v")
    commit = args.commit or run(["git", "rev-parse", "HEAD"])
    if not re.fullmatch(r"[0-9a-f]{7,40}", commit):
        parser.error("--commit must be a hexadecimal git revision")
    epoch = args.source_date_epoch
    if epoch is None:
        epoch = int(os.environ.get("SOURCE_DATE_EPOCH") or run(["git", "log", "-1", "--format=%ct"]))
    for path in release(args.version, commit, epoch, Path(args.out), tuple(args.target or TARGETS)):
        print(path)
    return 0


if __name__ == "__main__":
    sys.exit(main())
