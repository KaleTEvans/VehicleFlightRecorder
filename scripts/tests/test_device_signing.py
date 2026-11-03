import datetime
import importlib.util
from pathlib import Path
import unittest
spec = importlib.util.spec_from_file_location('device_signing', Path(__file__).resolve().parents[1] / 'device_signing.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class SigningTests(unittest.TestCase):
    def profile(self):
        return {'TeamIdentifier': ['TEAM'], 'UUID': 'fixture', 'ExpirationDate': datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(days=1),
                'Entitlements': {'application-identifier': 'TEAM.' + module.BUNDLE, 'get-task-allow': False}}
    def test_valid(self):
        self.assertEqual(module.export_options(self.profile())['method'], 'app-store-connect')
    def test_identity_expiry_and_distribution(self):
        for change in ['identity', 'expired', 'development', 'adhoc']:
            profile = self.profile()
            if change == 'identity': profile['Entitlements']['application-identifier'] = 'TEAM.other'
            if change == 'expired': profile['ExpirationDate'] = datetime.datetime(2000, 1, 1)
            if change == 'development': profile['Entitlements']['get-task-allow'] = True
            if change == 'adhoc': profile['ProvisionedDevices'] = ['fixture']
            with self.assertRaises(ValueError): module.export_options(profile)

class DeviceToolchainTests(unittest.TestCase):
    def test_device_profile_keeps_shared_pins(self):
        import sys
        sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
        from mobile_build_config import load_config
        preview = load_config(profile='preview')
        device = load_config(profile='device')
        self.assertEqual(device['MOBILE_XCODE_VERSION'], '26.0.1')
        self.assertEqual({k:v for k,v in preview.items() if k != 'MOBILE_XCODE_VERSION'},
                         {k:v for k,v in device.items() if k != 'MOBILE_XCODE_VERSION'})
