"""Validate explicit XCTest selection; targeted runs never claim full acceptance."""
import argparse
import json
import os
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
TARGETS = {'VehicleFlightRecorderTests', 'VehicleFlightRecorderUITests'}
SELECTOR = re.compile(r'VehicleFlightRecorder(?:UI)?Tests(?:/[A-Za-z_][A-Za-z0-9_]*){0,2}')
PASSED = re.compile(r"Test Case '-\[(VehicleFlightRecorder(?:UI)?Tests)\.(\w+) (test\w+)\]' passed")


def make_plan(scope, selectors='', *, root=ROOT, upload=False):
    if scope not in {'full', 'targeted'}:
        raise ValueError('Scope must be full or targeted')
    values = [x.strip() for x in selectors.split(',')] if selectors.strip() else []
    if scope == 'full' and values:
        raise ValueError('Full acceptance cannot filter tests')
    if scope == 'targeted' and (not values or upload):
        raise ValueError('Targeted runs require selectors and cannot upload to Appetize')
    if len(values) > 12 or len(values) != len(set(values)):
        raise ValueError('Use at most 12 unique selectors')
    for value in values:
        if len(value) > 240 or not SELECTOR.fullmatch(value):
            raise ValueError('Invalid XCTest selector')
        parts = value.split('/')
        directory = root/'ios'/parts[0]
        if not directory.is_dir():
            raise ValueError('Unknown XCTest target')
        if len(parts) > 1:
            file = directory/(parts[1]+'.swift')
            if not file.is_file() or not re.search(r'\bclass\s+'+re.escape(parts[1])+r'\s*:\s*XCTestCase\b', file.read_text()):
                raise ValueError('Unknown XCTest class')
            if len(parts) == 3 and not re.search(r'\bfunc\s+'+re.escape(parts[2])+r'\s*\(', file.read_text()):
                raise ValueError('Unknown XCTest method')
    return {'scope': scope, 'selectors': values, 'full_acceptance': scope == 'full'}


def verify_results(plan, log):
    passed = sorted({'/'.join(match) for match in PASSED.findall(log)})
    required = sorted(TARGETS) if plan['scope'] == 'full' else plan['selectors']
    for selector in required:
        if not any(test == selector or test.startswith(selector+'/') for test in passed):
            raise ValueError('No successful test evidence for '+selector)
    return dict(plan, status='passed', passed_tests=passed)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--github-output', type=Path)
    args = parser.parse_args()
    plan = make_plan(os.environ.get('NATIVE_TEST_SCOPE', 'full'), os.environ.get('NATIVE_TEST_SELECTORS', ''), upload=os.environ.get('UPLOAD_TO_APPETIZE') == 'true')
    if args.github_output:
        with args.github_output.open('a') as file:
            file.write('scope='+plan['scope']+'\n')
    print(json.dumps(plan))


if __name__ == '__main__':
    main()
