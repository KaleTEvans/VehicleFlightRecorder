"""Regress Apple's observed zero-exit validation rejection without network access."""
import os
from pathlib import Path
import plistlib
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]


class UploadTestFlightTests(unittest.TestCase):
    def setUp(self):
        cache = ROOT / '.cache/tmp'
        cache.mkdir(parents=True, exist_ok=True)
        self.temp = tempfile.TemporaryDirectory(dir=cache)
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        fake = self.root / 'xcrun'
        fake.write_text('#!/bin/sh\ncat "$FAKE_UPLOAD_TRANSCRIPT"\nexit "${FAKE_UPLOAD_STATUS:-0}"\n')
        fake.chmod(0o700)
        self.ipa = self.root / 'fixture.ipa'
        self.ipa.touch()

    def upload(self, transcript, status=0):
        log = self.root / 'transcript.txt'
        log.write_text(transcript)
        env = dict(os.environ, PATH=str(self.root) + os.pathsep + os.environ['PATH'],
                   TMPDIR=str(self.root), API_ID='FIXTURE', API_ISSUER='fixture-issuer',
                   API_PRIVATE_KEYS_DIR=str(self.root), FAKE_UPLOAD_TRANSCRIPT=str(log),
                   FAKE_UPLOAD_STATUS=str(status))
        result = subprocess.run(['bash', str(ROOT / 'scripts/upload_testflight.sh'), str(self.ipa)],
                                env=env, capture_output=True, text=True)
        self.assertEqual(list(self.root.glob('vfr-testflight.*')), [])
        return result

    def test_zero_exit_validation_rejection_fails(self):
        result = self.upload('UPLOAD FAILED with 1 error\nERROR: Validation failed (409) Invalid bundle.\n')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Apple rejected', result.stderr)

    def test_modern_positive_receipt_passes(self):
        self.assertEqual(self.upload('UPLOAD SUCCEEDED\nDelivery UUID: fixture\n').returncode, 0)

    def test_legacy_positive_receipt_passes(self):
        self.assertEqual(self.upload('No errors uploading fixture.ipa\n').returncode, 0)

    def test_unconfirmed_zero_exit_fails(self):
        result = self.upload('Started upload.\n')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('unconfirmed', result.stderr)

    def test_nonzero_with_success_text_fails(self):
        self.assertNotEqual(self.upload('UPLOAD SUCCEEDED\n', status=7).returncode, 0)

    def test_error_with_success_text_fails(self):
        self.assertNotEqual(self.upload('UPLOAD SUCCEEDED\nERROR: rejection\n').returncode, 0)

    def test_ipad_declares_all_multitasking_orientations(self):
        with (ROOT / 'ios/VehicleFlightRecorder/Resources/Info.plist').open('rb') as f:
            info = plistlib.load(f)
        required = {'UIInterfaceOrientationPortrait', 'UIInterfaceOrientationPortraitUpsideDown',
                    'UIInterfaceOrientationLandscapeLeft', 'UIInterfaceOrientationLandscapeRight'}
        self.assertEqual(set(info['UISupportedInterfaceOrientations~ipad']), required)


if __name__ == '__main__':
    unittest.main()
