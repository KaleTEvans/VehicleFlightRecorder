#!/usr/bin/env python3
"""Create a fresh simulator matching the selected Xcode SDK; bound every wait."""

import json
import os
from pathlib import Path
import subprocess
import sys


def run(*args, timeout=30, capture=False):
    print('+ ' + ' '.join(args), flush=True)
    return subprocess.run(args, check=True, text=True, timeout=timeout,
                          stdout=subprocess.PIPE if capture else None).stdout


def select_runtime(runtimes, sdk_version):
    for runtime in runtimes:
        if (runtime.get('isAvailable') and
                runtime.get('identifier', '').startswith('com.apple.CoreSimulator.SimRuntime.iOS-') and
                runtime.get('version') == sdk_version):
            return runtime['identifier']
    raise ValueError(f'No available iOS runtime matching selected Xcode SDK {sdk_version}')


def main():
    output = Path('build/ios-preview/simulator-startup')
    output.mkdir(parents=True, exist_ok=True)
    try:
        sdk = run('xcrun', '--sdk', 'iphonesimulator', '--show-sdk-version', capture=True).strip()
        runtimes = run('xcrun', 'simctl', 'list', 'runtimes', '-j', capture=True)
        (output / 'runtimes.json').write_text(runtimes)
        runtime = select_runtime(json.loads(runtimes)['runtimes'], sdk)
        # Avoid relying on pre-created device order or its saved boot/migration state.
        device = run('xcrun', 'simctl', 'create', 'VFR Native Acceptance',
                     'com.apple.CoreSimulator.SimDeviceType.iPhone-16-Pro', runtime,
                     timeout=60, capture=True).strip()
        (output / 'destination.txt').write_text(f'Runtime: {runtime}\nDevice: {device}\n')
        print(f'Selected fresh iPhone 16 Pro: {device}, runtime {runtime}', flush=True)
        run('xcrun', 'simctl', 'boot', device, timeout=60)
        run('xcrun', 'simctl', 'bootstatus', device, '-b', timeout=240)
        # Only expose a test destination after CoreSimulator reports boot completion.
        with open(os.environ['GITHUB_OUTPUT'], 'a') as target:
            target.write(f'id={device}\n')
        return 0
    except (subprocess.SubprocessError, ValueError, KeyError, OSError) as error:
        print(f'::error::Simulator startup failed: {error}', flush=True)
        (output / 'failure.txt').write_text(str(error) + '\n')
        for name, command in [
            ('devices.json', ('xcrun', 'simctl', 'list', 'devices', '-j')),
            ('processes.txt', ('ps', '-axo', 'pid,ppid,etime,command')),
        ]:
            try:
                (output / name).write_text(run(*command, timeout=15, capture=True))
            except (subprocess.SubprocessError, OSError) as diagnostic_error:
                (output / name).write_text(str(diagnostic_error) + '\n')
        return 1


if __name__ == '__main__':
    sys.exit(main())
