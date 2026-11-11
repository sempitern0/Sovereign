from pathlib import Path
import shutil
import tempfile
import unittest
import sys

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))
from validate_catalog import validate  # noqa: E402


class CatalogTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        for part in ("schemas", "catalog/apps", "profiles"):
            shutil.copytree(ROOT / part, self.root / part)
        shutil.copy(ROOT / "catalog/install-allowlist.yaml", self.root / "catalog/install-allowlist.yaml")

    def tearDown(self):
        self.temp.cleanup()

    def test_valid_catalog(self):
        self.assertEqual(validate(self.root), [])

    def test_allowlist_must_match_verified_catalog(self):
        path = self.root / "catalog/install-allowlist.yaml"
        path.write_text(path.read_text().replace("package_id: org.localsend.localsend_app", "package_id: org.evil.app"))
        self.assertTrue(any("package_id differs" in e for e in validate(self.root)))

    def test_verified_requires_scope_and_provenance(self):
        path = self.root / "catalog/apps/vlc.yaml"
        path.write_text(path.read_text().replace("    provenance: community\n", "", 1))
        self.assertTrue(any("scope (user|machine) and provenance" in e for e in validate(self.root)))

    def test_machine_scope_only_on_windows(self):
        path = self.root / "catalog/apps/vlc.yaml"
        path.write_text(path.read_text().replace("    scope: user\n    provenance: community", "    scope: machine\n    provenance: community", 1))
        self.assertTrue(any("machine scope is only supported on windows" in e for e in validate(self.root)))

    def test_allowlist_scope_must_match_catalog(self):
        path = self.root / "catalog/install-allowlist.yaml"
        path.write_text(path.read_text().replace("    scope: machine\n", "    scope: user\n", 1))
        self.assertTrue(any("scope differs" in e for e in validate(self.root)))

    def test_allowlist_rejects_unverified_entry(self):
        path = self.root / "catalog/install-allowlist.yaml"
        path.write_text(path.read_text() + """  - application: libreoffice
    upstream: https://www.libreoffice.org/
    platform: linux
    provider: flatpak
    package_id: org.libreoffice.LibreOffice
    source: https://flathub.org/apps/org.libreoffice.LibreOffice
    scope: user
    provenance: upstream
""")
        self.assertTrue(any("not verified" in e for e in validate(self.root)))

    def test_unknown_app_rejected(self):
        with (self.root / "profiles/essential.yaml").open("a") as out:
            out.write("  - unknown-app\n")
        self.assertTrue(any("missing application" in e for e in validate(self.root)))

    def test_unverified_package_rejected(self):
        with (self.root / "catalog/apps/localsend.yaml").open("a") as out:
            out.write("    package_id: invalid.package\n")
        self.assertTrue(any("package_id" in e for e in validate(self.root)))

    def test_invalid_provider_rejected(self):
        path = self.root / "catalog/apps/libreoffice.yaml"
        path.write_text(path.read_text().replace("provider: winget", "provider: apt", 1))
        self.assertTrue(any("not supported" in e for e in validate(self.root)))

    def test_missing_dependency_rejected(self):
        with (self.root / "catalog/apps/vlc.yaml").open("a") as out:
            out.write("depends_on: [missing-app]\n")
        self.assertTrue(any("missing dependency" in e for e in validate(self.root)))

    def test_dependency_cycle_rejected(self):
        with (self.root / "catalog/apps/vlc.yaml").open("a") as out:
            out.write("depends_on: [localsend]\n")
        with (self.root / "catalog/apps/localsend.yaml").open("a") as out:
            out.write("depends_on: [vlc]\n")
        self.assertTrue(any("dependency cycle" in e for e in validate(self.root)))

    def test_alternative_bad_default_rejected(self):
        with (self.root / "profiles/essential.yaml").open("a") as out:
            out.write("alternatives:\n  - id: reader\n    candidates: [localsend, vlc]\n    defaults:\n      linux: libreoffice\n")
        self.assertTrue(any("alternative" in e for e in validate(self.root)))

    def test_verified_requires_evidence(self):
        path = self.root / "catalog/apps/vlc.yaml"
        path.write_text(path.read_text().replace("status: pending", "status: verified", 1))
        self.assertTrue(any("verified requires" in e for e in validate(self.root)))


if __name__ == "__main__":
    unittest.main()
