#!/usr/bin/env python3
"""Write latest.json and notes.json for a Myna release.

The app reads them as release assets of the newest release:

    https://github.com/istoo/myna/releases/latest/download/latest.json
    https://github.com/istoo/myna/releases/latest/download/notes.json

latest.json describes the release: its version, notes, page, and every
asset with its size and SHA-256. notes.json keeps every release's notes
this file has seen, newest first, so What's new still lists the ones
before an update.
"""

import argparse
import hashlib
import json
import os


def digest(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--repo", required=True, help="owner/name")
    p.add_argument("--version", required=True, help="0.1.2892, no v")
    p.add_argument("--notes", required=True, help="markdown file")
    p.add_argument("--out", default="dist")
    p.add_argument("--previous", help="notes.json of the last release")
    p.add_argument("assets", nargs="*")
    a = p.parse_args()

    tag = "v" + a.version.lstrip("v")
    page = f"https://github.com/{a.repo}/releases/tag/{tag}"
    notes = open(a.notes, encoding="utf-8").read()

    assets = {}
    for path in a.assets:
        name = os.path.basename(path)
        assets[name] = {
            "url": f"https://github.com/{a.repo}/releases/download/{tag}/{name}",
            "size": os.path.getsize(path),
            "sha256": digest(path),
        }

    feed = {
        "version": a.version.lstrip("v"),
        "notes": notes,
        "url": page,
        "assets": assets,
    }
    os.makedirs(a.out, exist_ok=True)
    with open(os.path.join(a.out, "latest.json"), "w", encoding="utf-8") as f:
        json.dump(feed, f, ensure_ascii=False, indent=2)

    releases = []
    if a.previous and os.path.exists(a.previous):
        try:
            old = json.load(open(a.previous, encoding="utf-8"))
            releases = [
                r for r in old.get("releases", [])
                if r.get("version") != feed["version"]
            ]
        except (ValueError, OSError):
            releases = []
    releases.insert(0, {"version": feed["version"], "notes": notes, "url": page})
    with open(os.path.join(a.out, "notes.json"), "w", encoding="utf-8") as f:
        json.dump({"releases": releases}, f, ensure_ascii=False, indent=2)

    print(f"latest.json: {feed['version']}, {len(assets)} assets")
    print(f"notes.json: {len(releases)} releases")


if __name__ == "__main__":
    main()