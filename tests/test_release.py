from pathlib import Path
import hashlib
import shutil
import subprocess
import sys
import tarfile
import tempfile
import unittest
import zipfile

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))
import release  # noqa: E402

GO_VERSION_M = (
    "/tmp/suitectl.exe: go1.24.7\n"
    "\tpath\tgithub.com/sempitern0/Sovereign/cmd/suitectl\n"
    "\tmod\tgithub.com/sempitern0/Sovereign\t(devel)\t\n"
    "\tdep\tgopkg.in/yaml.v3\tv3.0.1\t\n"
    "\tbuild\t-trimpath=true\n"
)

MEMBERS = [("pkg/suitectl", b"\x7fELF binary", 0o755), ("pkg/README.md", b"readme", 0o644)]


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.dir = Path(self.temp.name)

    def tearDown(self):
        self.temp.cleanup()

    def test_semver_and_names(self):
        for good in ("0.3.0", "1.2.3-rc.1", "10.0.0-pr.abc123"):
            self.assertRegex(good, release.SEMVER)
        for bad in ("v0.3.0", "0.3", "01.2.3", "0.3.0 ; rm -rf /"):
            self.assertNotRegex(bad, release.SEMVER)
        self.assertEqual(release.artifact_base("0.3.0", "windows/arm64"), "suitectl_0.3.0_windows_arm64")

    def test_tar_gz_is_deterministic_with_fixed_metadata(self):
        a, b = self.dir / "a.tar.gz", self.dir / "b.tar.gz"
        release.write_tar_gz(a, MEMBERS, 1763000000)
        release.write_tar_gz(b, MEMBERS, 1763000000)
        self.assertEqual(a.read_bytes(), b.read_bytes())
        header = a.read_bytes()[:10]
        self.assertEqual(int.from_bytes(header[4:8], "little"), 1763000000)  # gzip MTIME, no wall clock
        self.assertEqual(header[3] & 0x08, 0)  # no FNAME: temporary paths never leak
        with tarfile.open(a) as tar:
            members = {m.name: m for m in tar.getmembers()}
        self.assertEqual(members["pkg/suitectl"].mode, 0o755)
        self.assertEqual(members["pkg/README.md"].mode, 0o644)
        self.assertTrue(all(m.uid == 0 and m.gid == 0 and m.mtime == 1763000000 for m in members.values()))

    def test_zip_is_deterministic_and_keeps_modes(self):
        a, b = self.dir / "a.zip", self.dir / "b.zip"
        release.write_zip(a, MEMBERS, 1763000000)
        release.write_zip(b, MEMBERS, 1763000000)
        self.assertEqual(a.read_bytes(), b.read_bytes())
        with zipfile.ZipFile(a) as archive:
            modes = {i.filename: (i.external_attr >> 16) & 0o777 for i in archive.infolist()}
            self.assertEqual(archive.read("pkg/suitectl"), b"\x7fELF binary")
        self.assertEqual(modes, {"pkg/suitectl": 0o755, "pkg/README.md": 0o644})

    def test_archive_members_are_sorted_and_include_docs(self):
        binary = self.dir / "suitectl"
        binary.write_bytes(b"bin")
        members = release.archive_members(binary, "suitectl_0.3.0_linux_amd64")
        names = [name for name, _, _ in members]
        self.assertEqual(names, sorted(names))
        self.assertEqual({n.split("/")[1] for n in names}, {"suitectl", *release.DOCS})
        self.assertEqual(dict((n, m) for n, _, m in members)["suitectl_0.3.0_linux_amd64/suitectl"], 0o755)

    def test_zip_clamps_pre_1980_timestamps(self):
        release.write_zip(self.dir / "old.zip", MEMBERS, 0)
        with zipfile.ZipFile(self.dir / "old.zip") as archive:
            self.assertEqual(archive.infolist()[0].date_time[0], 1980)

    def test_parse_go_version_m(self):
        info = release.parse_go_version_m(GO_VERSION_M)
        self.assertEqual(info, {"go": "go1.24.7", "deps": [{"path": "gopkg.in/yaml.v3", "version": "v3.0.1"}]})

    def test_sbom_is_spdx_with_dependencies(self):
        build = release.parse_go_version_m(GO_VERSION_M)
        sbom = release.build_sbom("suitectl_0.3.0_linux_amd64", "0.3.0", "suitectl", "ab" * 32, build, 1763000000)
        self.assertEqual(sbom["spdxVersion"], "SPDX-2.3")
        self.assertEqual(sbom["creationInfo"]["created"], "2025-11-13T02:13:20Z")
        names = [p["name"] for p in sbom["packages"]]
        self.assertEqual(names, ["suitectl", "golang/stdlib", "gopkg.in/yaml.v3"])
        self.assertEqual(sbom["packages"][0]["checksums"][0], {"algorithm": "SHA256", "checksumValue": "ab" * 32})
        ids = {p["SPDXID"] for p in sbom["packages"]}
        self.assertEqual(len(ids), 3)
        depends = [r for r in sbom["relationships"] if r["relationshipType"] == "DEPENDS_ON"]
        self.assertEqual(len(depends), 2)
        self.assertTrue(all(r["relatedSpdxElement"] in ids for r in sbom["relationships"]))

    @unittest.skipUnless(shutil.which("go"), "Go toolchain not available")
    def test_release_build_is_reproducible(self):
        commit = subprocess.run(["git", "rev-parse", "HEAD"], cwd=ROOT, capture_output=True, text=True).stdout.strip()
        if not commit:
            self.skipTest("not a git checkout")
        sums = []
        for name in ("one", "two"):
            out = self.dir / name
            release.release("0.0.0-test", commit, 1763000000, out, ("linux/amd64",))
            sums.append((out / "SHA256SUMS").read_text())
            for line in sums[-1].splitlines():
                digest, file = line.split("  ")
                self.assertEqual(hashlib.sha256((out / file).read_bytes()).hexdigest(), digest)
        self.assertEqual(sums[0], sums[1])


if __name__ == "__main__":
    unittest.main()
