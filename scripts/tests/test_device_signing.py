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
