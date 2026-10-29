"""Verify device/simulator ARM64 slices and retain native build provenance."""
import argparse
import hashlib
import json
from pathlib import Path
import plistlib
import subprocess
import sys

try:
    from .mobile_build_config import ROOT, load_config
except ImportError:
    from mobile_build_config import ROOT, load_config


def inspect_framework(path):
    with (path / 'Info.plist').open('rb') as file:
        libraries = plistlib.load(file)['AvailableLibraries']
    found = set()
    for library in libraries:
        variant = library.get('SupportedPlatformVariant', 'device')
        if (library['SupportedPlatform'] != 'ios' or variant not in ('device', 'simulator') or
                library['SupportedArchitectures'] != ['arm64'] or variant in found):
            raise ValueError('Unexpected or duplicate native framework slice')
        found.add(variant)
        directory = path / library['LibraryIdentifier'] / library['LibraryPath']
        if not (directory / 'VehicleFlightRecorderCore').is_file():
            raise ValueError('Missing framework binary')
    if found != {'device', 'simulator'}:
        raise ValueError('Both device and simulator ARM64 slices are required')
    return libraries


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('framework', type=Path)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if sys.platform != 'darwin':
        parser.error('Native binary verification requires macOS; metadata tests are separate')
    config = load_config()
    libraries = inspect_framework(args.framework)
    for library in libraries:
        binary = args.framework / library['LibraryIdentifier'] / library['LibraryPath'] / 'VehicleFlightRecorderCore'
        subprocess.run(['xcrun', 'lipo', str(binary), '-verify_arch', 'arm64'], check=True)
    inputs = sorted((ROOT / 'core').rglob('*.go')) + [ROOT / 'core/go.mod', ROOT / 'core/go.sum', ROOT / 'scripts/mobile_toolchain.env', ROOT / 'scripts/build_go_ios.sh', ROOT / 'ios/project.yml']
    inputs += [ROOT / 'scripts/device_toolchain.env', ROOT / 'scripts/mobile_build_config.py']
    inputs += sorted((ROOT / 'vehicle-packs').rglob('*.json'))
    report = {'configuration': config, 'slices': libraries,
              'commit': subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip(),
              'go': subprocess.check_output(['go', 'version'], text=True).strip(),
              'xcode': subprocess.check_output(['xcodebuild', '-version'], text=True).strip(),
              'sdk_versions': {sdk: subprocess.check_output(['xcrun', '--sdk', sdk, '--show-sdk-version'], text=True).strip() for sdk in ['iphoneos', 'iphonesimulator']},
              'source_sha256': {str(file.relative_to(ROOT)): hashlib.sha256(file.read_bytes()).hexdigest() for file in inputs}}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report, indent=2))


if __name__ == '__main__':
    main()
