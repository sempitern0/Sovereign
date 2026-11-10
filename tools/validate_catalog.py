#!/usr/bin/env python3
"""Validate Sovereign catalog and profiles without host modifications."""
import argparse
import datetime as dt
import json
from pathlib import Path
import sys

import jsonschema
import yaml

PLATFORM_PROVIDERS = {
    "windows": {"winget", "manual", "none"},
    "linux": {"flatpak", "apt", "dnf", "manual", "none"},
    "android": {"fdroid", "obtainium", "manual", "none"},
}


def load(path: Path) -> dict:
    with path.open(encoding="utf-8") as handle:
        result = yaml.safe_load(handle)
    if not isinstance(result, dict):
        raise ValueError("document must be a mapping")
    return result


def validate(root: Path) -> list[str]:
    errors: list[str] = []
    apps: set[str] = set()
    schemas = {}
    for kind in ("application", "profile"):
        schemas[kind] = json.loads((root / "schemas" / f"{kind}.schema.json").read_text(encoding="utf-8"))
        jsonschema.Draft202012Validator.check_schema(schemas[kind])

    for kind, folder in (("application", "catalog/apps"), ("profile", "profiles")):
        paths = sorted((root / folder).glob("*.yaml"))
        if not paths:
            errors.append(f"{folder}: at least one YAML manifest is required")
        for path in paths:
            try:
                obj = load(path)
                jsonschema.Draft202012Validator(schemas[kind], format_checker=jsonschema.FormatChecker()).validate(obj)
                if obj["id"] != path.stem:
                    raise ValueError("manifest id differs from filename")
                if kind == "application":
                    for platform, config in obj["platforms"].items():
                        status, provider = config["status"], config["provider"]
                        if provider not in PLATFORM_PROVIDERS[platform]:
                            raise ValueError(f"{platform}: provider {provider!r} not supported")
                        if status == "unsupported" and provider != "none":
                            raise ValueError(f"{platform}: unsupported requires provider none")
                        if status == "verified":
                            if provider == "none" or not all(config.get(field) for field in ("package_id", "source", "verified_at")):
                                raise ValueError(f"{platform}: verified requires package_id, source and verified_at")
                            if dt.date.fromisoformat(config["verified_at"]) > dt.date.today():
                                raise ValueError(f"{platform}: verified_at is in the future")
                        if status != "verified" and "package_id" in config:
                            raise ValueError(f"{platform}: unverified entries cannot advertise package_id")
                    apps.add(obj["id"])
            except (ValueError, yaml.YAMLError, jsonschema.ValidationError) as exc:
                errors.append(f"{path.relative_to(root)}: {exc}")
    for path in sorted((root / "profiles").glob("*.yaml")):
        try:
            profile = load(path)
            for app_id in profile.get("applications", []) + profile.get("optional", []):
                if app_id not in apps:
                    errors.append(f"{path.relative_to(root)}: missing application {app_id!r}")
            if set(profile.get("applications", [])) & set(profile.get("optional", [])):
                errors.append(f"{path.relative_to(root)}: required and optional lists overlap")
        except (ValueError, yaml.YAMLError) as exc:
            errors.append(f"{path.relative_to(root)}: {exc}")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()
    errors = validate(args.root.resolve())
    for error in errors:
        print("ERROR:", error, file=sys.stderr)
    if errors:
        return 1
    print("Catalog and profiles validated; no packages are enabled for installation.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
