"""Run XCTest once and retain evidence for every requested target/class/method."""
import json
import os
from pathlib import Path
import re
import subprocess
import sys

try:
    from .native_test_plan import ROOT, make_plan, verify_results
except ImportError:
    from native_test_plan import ROOT, make_plan, verify_results


def command(simulator, plan):
    if not re.fullmatch(r'[0-9A-Fa-f]{8}(?:-[0-9A-Fa-f]{4}){3}-[0-9A-Fa-f]{12}', simulator):
        raise ValueError('Invalid simulator identity')
    return ['xcodebuild', '-project', 'ios/VehicleFlightRecorder.xcodeproj',
            '-scheme', 'VehicleFlightRecorder', '-configuration', 'Debug',
            '-destination', 'platform=iOS Simulator,id='+simulator,
            '-derivedDataPath', 'build/ios-preview/DerivedData',
            '-resultBundlePath', 'build/ios-preview/NativeTests.xcresult',
            '-parallel-testing-enabled', 'NO',
            *['-only-testing:'+value for value in plan['selectors']],
            'ARCHS=arm64', 'ONLY_ACTIVE_ARCH=YES', 'CODE_SIGNING_ALLOWED=NO', 'test']


def main():
    plan = make_plan(os.environ.get('NATIVE_TEST_SCOPE', 'full'), os.environ.get('NATIVE_TEST_SELECTORS', ''), upload=os.environ.get('UPLOAD_TO_APPETIZE') == 'true')
    args = command(os.environ.get('SIMULATOR_ID', ''), plan)
    output = ROOT/'build/native-acceptance'
    output.mkdir(parents=True, exist_ok=True)
    report = output/'native-test-scope.json'
    report.write_text(json.dumps(dict(plan, status='running'), indent=2)+'\n')
    lines = []
    with (output/'xcodebuild-test.log').open('w') as log:
        with subprocess.Popen(args, cwd=ROOT, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, errors='replace') as process:
            for line in process.stdout:
                sys.stdout.write(line); sys.stdout.flush(); log.write(line); lines.append(line)
            code = process.wait()
    if code:
        report.write_text(json.dumps(dict(plan, status='failed', exit_code=code), indent=2)+'\n')
        raise SystemExit(code)
    try:
        result = verify_results(plan, ''.join(lines))
    except ValueError:
        report.write_text(json.dumps(dict(plan, status='missing_test_evidence'), indent=2)+'\n')
        raise
    report.write_text(json.dumps(result, indent=2)+'\n')
    print(f"Verified {len(result['passed_tests'])} successful tests; scope={plan['scope']}")


if __name__ == '__main__':
    main()
