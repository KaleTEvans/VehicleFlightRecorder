"""Validate and package the real Xcode simulator product; requires macOS ditto."""

import argparse
import os
from pathlib import Path
import plistlib
import subprocess
import sys
import zipfile


def discover_app(products: Path) -> Path:
    apps = sorted(path for path in products.glob("*.app") if path.is_dir())
    if len(apps) != 1:
        raise ValueError(f"Expected exactly one simulator .app in {products}; found {len(apps)}")
    app = apps[0]
    with (app / "Info.plist").open("rb") as source:
        info = plistlib.load(source)
    expected = {
        "CFBundleIdentifier": "com.kaleevans.vehicleflightrecorder",
        "CFBundleDisplayName": "Vehicle Flight Recorder",
        "CFBundleShortVersionString": "0.1.0",
        "CFBundlePackageType": "APPL",
        "CFBundleSupportedPlatforms": ["iPhoneSimulator"],
    }
    for key, value in expected.items():
        if info.get(key) != value:
            raise ValueError(f"Unexpected {key}: {info.get(key)!r}; expected {value!r}")
    executable = info.get("CFBundleExecutable", "")
    if not executable or Path(executable).name != executable:
        raise ValueError("Missing or invalid CFBundleExecutable")
    binary = app / executable
    if not binary.is_file() or not os.access(binary, os.X_OK):
        raise ValueError(f"Missing executable simulator binary: {binary}")
    return app


def verify_archive(archive: Path, app_name: str) -> None:
    with zipfile.ZipFile(archive) as bundle:
        names = bundle.namelist()
        if not names or {name.split("/")[0] for name in names} != {app_name}:
            raise ValueError("ZIP must contain only the .app bundle at ZIP root")
        if f"{app_name}/Info.plist" not in names:
            raise ValueError("ZIP is missing the app Info.plist")
        if bundle.testzip() is not None:
            raise ValueError("ZIP integrity check failed")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("products", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    if sys.platform != "darwin":
        parser.error("Packaging requires a real Xcode simulator build and macOS ditto; run the iOS Simulator Preview workflow.")
    app = discover_app(args.products.resolve())
    with (app / "Info.plist").open("rb") as source:
        executable = plistlib.load(source)["CFBundleExecutable"]
    subprocess.run(["xcrun", "lipo", str(app / executable), "-verify_arch", "arm64"], check=True)
    output = args.output.resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    if output.exists():
        raise ValueError(f"Refusing to reuse an existing archive: {output}")
    # ditto preserves native executable permissions and framework symlinks.
    subprocess.run(["ditto", "-c", "-k", "--norsrc", "--keepParent", str(app), str(output)], check=True)
    verify_archive(output, app.name)
    print(f"Verified ARM simulator bundle: {app.name}")
    print(f"Packaged: {output}")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError, subprocess.CalledProcessError, zipfile.BadZipFile) as error:
        sys.exit(f"Simulator packaging failed: {error}")
