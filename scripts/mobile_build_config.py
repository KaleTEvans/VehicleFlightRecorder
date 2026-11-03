"""Read/validate the single native toolchain contract without needing Go or Xcode."""
import argparse
import os
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]


def load_config(root=ROOT, profile=None):
    config = {}
    for line in (root / 'scripts/mobile_toolchain.env').read_text().splitlines():
        if not line or line.startswith('#'):
            continue
        key, value = line.split('=', 1)
        config[key] = value
    profile = profile if profile is not None else os.environ.get('MOBILE_BUILD_PROFILE', 'preview')
    if profile not in ('preview', 'device'):
        raise ValueError('Unknown native build profile')
    if profile == 'device':
        overlay = (root / 'scripts/device_toolchain.env').read_text().splitlines()
        values = dict(line.split('=', 1) for line in overlay if line and not line.startswith('#'))
        if set(values) != {'MOBILE_XCODE_VERSION'} or not re.fullmatch(r'26\.\d+(?:\.\d+)?', values['MOBILE_XCODE_VERSION']):
            raise ValueError('Invalid signed-device Xcode contract')
        config.update(values)
    formats = {
        'MOBILE_GO_VERSION': r'go1\.\d+\.\d+',
        'MOBILE_VERSION': r'v\d+\.\d+\.\d+-\d{14}-[0-9a-f]{12}',
        'MOBILE_XCODE_VERSION': r'\d+\.\d+(?:\.\d+)?',
        'MOBILE_IOS_VERSION': r'\d+\.\d+',
        'MOBILE_TARGETS': r'ios/arm64,iossimulator/arm64',
    }
    if set(config) != set(formats) or any(not re.fullmatch(pattern, config[key]) for key, pattern in formats.items()):
        raise ValueError('Invalid/incomplete native toolchain configuration')
    if tuple(map(int, config['MOBILE_GO_VERSION'][2:].split('.'))) < (1, 26, 0):
        raise ValueError('Native Go compiler lacks the iOS cgo alignment fix')
    module = (root / 'core/go.mod').read_text()
    if not re.search(r'golang.org/x/mobile\s+' + re.escape(config['MOBILE_VERSION']) + r'\s', module):
        raise ValueError('gomobile/gobind/runtime module pin mismatch')
    if re.search(r'^replace\s', module, re.M):
        raise ValueError('Native build does not accept unverified module replacements')
    project = (root / 'ios/project.yml').read_text()
    if f'iOS: "{config["MOBILE_IOS_VERSION"]}"' not in project:
        raise ValueError('Native deployment floor does not match XcodeGen project')
    return config


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--github-output', type=Path)
    parser.add_argument('--github-env', type=Path)
    args = parser.parse_args()
    config = load_config()
    print(config)
    if args.github_output:
        with args.github_output.open('a') as file:
            file.write('go_version=' + config['MOBILE_GO_VERSION'][2:] + '\n')
    if args.github_env:
        with args.github_env.open('a') as file:
            file.write('DEVELOPER_DIR=/Applications/Xcode_' + config['MOBILE_XCODE_VERSION'] + '.app/Contents/Developer\n')


if __name__ == '__main__':
    main()
