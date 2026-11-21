import hashlib
import importlib.util
from pathlib import Path
import plistlib
import tempfile
import unittest
import zipfile

ROOT=Path(__file__).resolve().parents[2]
spec=importlib.util.spec_from_file_location('verify_prepared_ipa',ROOT/'scripts/verify_prepared_ipa.py')
module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)

class PreparedIPATests(unittest.TestCase):
    def setUp(self):
        cache=ROOT/'.cache/tmp';cache.mkdir(parents=True,exist_ok=True)
        temp=tempfile.TemporaryDirectory(dir=cache);self.addCleanup(temp.cleanup);self.root=Path(temp.name)
        self.ipa=self.root/'VehicleFlightRecorder.ipa';self.commit='1'*40;self.build='14.1'
        self.write_ipa()
        (self.root/'provenance.txt').write_text(f'Commit: {self.commit}\nBuild: {self.build}\n')
    def write_ipa(self,bundle='com.kaleevans.vehicleflightrecorder',build='14.1'):
        with zipfile.ZipFile(self.ipa,'w') as archive:archive.writestr('Payload/VehicleFlightRecorder.app/Info.plist',plistlib.dumps(dict(CFBundleIdentifier=bundle,CFBundleVersion=build,CFBundleShortVersionString='0.1.0')))
    def verify(self,sha=None):
        return module.verify(self.root,sha or hashlib.sha256(self.ipa.read_bytes()).hexdigest(),self.commit,self.build)
    def test_exact_payload_and_provenance_pass(self):self.assertEqual(self.verify(),self.ipa)
    def test_changed_payload_rejected(self):
        sha=hashlib.sha256(self.ipa.read_bytes()).hexdigest();self.write_ipa(build='14.2')
        with self.assertRaises(ValueError):self.verify(sha)
    def test_wrong_destination_or_build_rejected_even_with_matching_digest(self):
        for bundle,build in [('com.other.app','14.1'),('com.kaleevans.vehicleflightrecorder','14.2')]:
            self.write_ipa(bundle,build)
            with self.assertRaises(ValueError):self.verify()
    def test_wrong_source_provenance_rejected(self):
        (self.root/'provenance.txt').write_text('Commit: '+'2'*40+'\nBuild: 14.1\n')
        with self.assertRaises(ValueError):self.verify()
    def test_ambiguous_or_injected_identity_rejected(self):
        (self.root/'second.ipa').write_bytes(self.ipa.read_bytes())
        with self.assertRaises(ValueError):self.verify()
        with self.assertRaises(ValueError):module.verify(self.root,'$(false)',self.commit,self.build)
