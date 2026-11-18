#!/usr/bin/env python3
"""Verify the exact retained signed payload before an explicitly approved upload."""
import argparse
import hashlib
from pathlib import Path
import plistlib
import re
import zipfile


def verify(directory, expected_sha256, expected_commit, expected_build):
    if not re.fullmatch(r'[0-9a-f]{64}', expected_sha256) or not re.fullmatch(r'[0-9a-f]{40}', expected_commit) or not re.fullmatch(r'[0-9]+\.[0-9]+', expected_build):
        raise ValueError('Exact SHA-256, source commit and build are required')
    root=Path(directory).resolve()
    ipas=list(root.rglob('*.ipa'));provenance=list(root.rglob('provenance.txt'))
    if len(ipas)!=1 or len(provenance)!=1: raise ValueError('Expected one retained IPA and one provenance record')
    ipa=ipas[0]
    for file in (ipa,provenance[0]):
        if file.is_symlink() or not file.resolve().is_relative_to(root): raise ValueError('Nonlocal prepared artifact')
    if ipa.stat().st_size>128*1024*1024: raise ValueError('Prepared IPA exceeds supported payload size')
    digest=hashlib.sha256(ipa.read_bytes()).hexdigest()
    if digest!=expected_sha256: raise ValueError('Prepared IPA differs from the approved checksum')
    lines=provenance[0].read_text().splitlines()
    if f'Commit: {expected_commit}' not in lines or f'Build: {expected_build}' not in lines: raise ValueError('Prepared source/build provenance mismatch')
    with zipfile.ZipFile(ipa) as archive:
        name='Payload/VehicleFlightRecorder.app/Info.plist'
        if archive.namelist().count(name)!=1 or archive.getinfo(name).file_size>1024*1024: raise ValueError('Invalid prepared app metadata')
        info=plistlib.loads(archive.read(name))
    if (info.get('CFBundleIdentifier'),info.get('CFBundleShortVersionString'),info.get('CFBundleVersion'))!=('com.kaleevans.vehicleflightrecorder','0.1.0',expected_build):
        raise ValueError('Prepared IPA does not match the approved application/version/build')
    return ipa


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('directory');parser.add_argument('--sha256',required=True);parser.add_argument('--commit',required=True);parser.add_argument('--build',required=True);parser.add_argument('--github-output')
    args=parser.parse_args()
    try:ipa=verify(args.directory,args.sha256,args.commit,args.build)
    except (ValueError,OSError,zipfile.BadZipFile,plistlib.InvalidFileException) as error:raise SystemExit(str(error))
    if args.github_output:
        with open(args.github_output,'a') as output:output.write(f'ipa_path={ipa}\n')
    print(f'Verified retained {args.build} IPA, source {args.commit}, SHA-256 {args.sha256}')

if __name__=='__main__':main()
