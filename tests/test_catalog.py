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

    def tearDown(self):
        self.temp.cleanup()

    def test_valid_catalog(self):
        self.assertEqual(validate(self.root), [])

    def test_unknown_app_rejected(self):
        with (self.root / "profiles/essential.yaml").open("a") as out:
            out.write("  - unknown-app\n")
        self.assertTrue(any("missing application" in e for e in validate(self.root)))

    def test_unverified_package_rejected(self):
        with (self.root / "catalog/apps/localsend.yaml").open("a") as out:
            out.write("    package_id: invalid.package\n")
        self.assertTrue(any("package_id" in e for e in validate(self.root)))

    def test_invalid_provider_rejected(self):
        path = self.root / "catalog/apps/vlc.yaml"
        path.write_text(path.read_text().replace("provider: winget", "provider: apt", 1))
        self.assertTrue(any("not supported" in e for e in validate(self.root)))

    def test_verified_requires_evidence(self):
        path = self.root / "catalog/apps/vlc.yaml"
        path.write_text(path.read_text().replace("status: pending", "status: verified", 1))
        self.assertTrue(any("verified requires" in e for e in validate(self.root)))


if __name__ == "__main__":
    unittest.main()
