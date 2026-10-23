"""Synthetic metadata/archive tests only; these do not build or launch iOS."""

from pathlib import Path
import plistlib
import tempfile
import unittest
import zipfile

from scripts.package_ios_preview import discover_app, verify_archive


class SimulatorPackageTests(unittest.TestCase):
    def setUp(self):
        self.workspace = tempfile.TemporaryDirectory()
        self.addCleanup(self.workspace.cleanup)
        self.products = Path(self.workspace.name)
        self.app = self.products / "VehicleFlightRecorder.app"
        self.app.mkdir()
        self.info = {
            "CFBundleIdentifier": "com.kaleevans.vehicleflightrecorder",
            "CFBundleDisplayName": "Vehicle Flight Recorder",
            "CFBundleShortVersionString": "0.1.0",
            "CFBundlePackageType": "APPL",
            "CFBundleSupportedPlatforms": ["iPhoneSimulator"],
            "CFBundleExecutable": "VehicleFlightRecorder",
        }
        self.write_info()
        binary = self.app / "VehicleFlightRecorder"
        binary.write_text("synthetic test fixture, not an iOS binary\n")
        binary.chmod(0o755)

    def write_info(self):
        with (self.app / "Info.plist").open("wb") as output:
            plistlib.dump(self.info, output)

    def test_expected_simulator_metadata(self):
        self.assertEqual(discover_app(self.products), self.app)

    def test_rejects_device_build(self):
        self.info["CFBundleSupportedPlatforms"] = ["iPhoneOS"]
        self.write_info()
        with self.assertRaisesRegex(ValueError, "CFBundleSupportedPlatforms"):
            discover_app(self.products)

    def test_rejects_wrong_identity(self):
        self.info["CFBundleIdentifier"] = "example.unapproved"
        self.write_info()
        with self.assertRaisesRegex(ValueError, "CFBundleIdentifier"):
            discover_app(self.products)

    def test_rejects_multiple_apps(self):
        (self.products / "Other.app").mkdir()
        with self.assertRaisesRegex(ValueError, "exactly one"):
            discover_app(self.products)

    def test_rejects_missing_app(self):
        with self.assertRaisesRegex(ValueError, "found 0"):
            discover_app(self.products / "missing")

    def test_rejects_missing_executable(self):
        (self.app / "VehicleFlightRecorder").unlink()
        with self.assertRaisesRegex(ValueError, "Missing executable"):
            discover_app(self.products)

    def test_archive_requires_app_at_root(self):
        archive = self.products / "bundle.zip"
        with zipfile.ZipFile(archive, "w") as output:
            output.writestr(f"{self.app.name}/Info.plist", b"fixture")
        verify_archive(archive, self.app.name)
        with zipfile.ZipFile(archive, "w") as output:
            output.writestr(f"Debug-iphonesimulator/{self.app.name}/Info.plist", b"fixture")
        with self.assertRaisesRegex(ValueError, "ZIP root"):
            verify_archive(archive, self.app.name)
