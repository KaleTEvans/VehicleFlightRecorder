"""Synthetic configuration/metadata validation; not native compilation evidence."""
from pathlib import Path
import plistlib
import tempfile
import unittest

from scripts.mobile_build_config import ROOT, load_config
from scripts.verify_ios_framework import inspect_framework


class NativeBuildContractTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for name in ['scripts/mobile_toolchain.env', 'core/go.mod', 'ios/project.yml']:
            target = self.root / name
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes((ROOT / name).read_bytes())

    def test_single_source_configuration(self):
        self.assertEqual(load_config(self.root)['MOBILE_TARGETS'], 'ios/arm64,iossimulator/arm64')

    def test_rejects_runtime_generator_mismatch(self):
        module = self.root / 'core/go.mod'
        module.write_text(module.read_text().replace('6912353760cf', '000000000000'))
        with self.assertRaisesRegex(ValueError, 'pin mismatch'):
            load_config(self.root)

    def test_rejects_old_native_compiler(self):
        path = self.root / 'scripts/mobile_toolchain.env'
        path.write_text(path.read_text().replace('go1.26.10', 'go1.24.0'))
        with self.assertRaisesRegex(ValueError, 'alignment fix'):
            load_config(self.root)

    def test_rejects_deployment_floor_mismatch(self):
        path = self.root / 'ios/project.yml'
        path.write_text(path.read_text().replace('iOS: "17.0"', 'iOS: "18.0"'))
        with self.assertRaisesRegex(ValueError, 'deployment floor'):
            load_config(self.root)

    def test_rejects_unverified_replacement(self):
        path = self.root / 'core/go.mod'
        path.write_text(path.read_text() + '\nreplace golang.org/x/mobile => ../other\n')
        with self.assertRaisesRegex(ValueError, 'replacements'):
            load_config(self.root)

    def framework(self):
        libraries = []
        for variant in ['device', 'simulator']:
            identifier = 'ios-arm64' + ('-simulator' if variant == 'simulator' else '')
            library = {'LibraryIdentifier': identifier, 'LibraryPath': 'VehicleFlightRecorderCore.framework',
                       'SupportedPlatform': 'ios', 'SupportedArchitectures': ['arm64']}
            if variant == 'simulator':
                library['SupportedPlatformVariant'] = variant
            libraries.append(library)
            path = self.root / identifier / library['LibraryPath']
            path.mkdir(parents=True)
            (path / 'VehicleFlightRecorderCore').write_bytes(b'synthetic metadata fixture')
        return libraries

    def write_libraries(self, libraries):
        with (self.root / 'Info.plist').open('wb') as file:
            plistlib.dump({'AvailableLibraries': libraries}, file)

    def test_both_correct_slice_metadata(self):
        self.write_libraries(self.framework())
        self.assertEqual(len(inspect_framework(self.root)), 2)

    def test_simulator_only_is_not_device_preparation(self):
        self.write_libraries(self.framework()[1:])
        with self.assertRaisesRegex(ValueError, 'Both device and simulator'):
            inspect_framework(self.root)

    def test_wrong_architecture_rejected(self):
        libraries = self.framework()
        libraries[0]['SupportedArchitectures'] = ['x86_64']
        self.write_libraries(libraries)
        with self.assertRaisesRegex(ValueError, 'Unexpected'):
            inspect_framework(self.root)

    def test_missing_binary_rejected(self):
        libraries = self.framework()
        (self.root / libraries[0]['LibraryIdentifier'] / libraries[0]['LibraryPath'] / 'VehicleFlightRecorderCore').unlink()
        self.write_libraries(libraries)
        with self.assertRaisesRegex(ValueError, 'Missing framework binary'):
            inspect_framework(self.root)
