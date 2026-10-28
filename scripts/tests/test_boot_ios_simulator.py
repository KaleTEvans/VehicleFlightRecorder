"""Selection/timeout checks with synthetic inputs; no iOS execution in WSL."""
import subprocess
import unittest
from unittest.mock import patch

from scripts.boot_ios_simulator import run, select_runtime


class SimulatorStartupTests(unittest.TestCase):
    def test_uses_sdk_runtime_instead_of_first_listed(self):
        runtimes = [
            {'identifier': 'com.apple.CoreSimulator.SimRuntime.iOS-18-0', 'version': '18.0', 'isAvailable': True},
            {'identifier': 'com.apple.CoreSimulator.SimRuntime.iOS-18-5', 'version': '18.5', 'isAvailable': True},
        ]
        self.assertEqual(select_runtime(runtimes, '18.5'), runtimes[1]['identifier'])

    def test_unavailable_matching_runtime_fails_explicitly(self):
        with self.assertRaisesRegex(ValueError, 'matching selected Xcode SDK'):
            select_runtime([{'identifier': 'com.apple.CoreSimulator.SimRuntime.iOS-18-5',
                             'version': '18.5', 'isAvailable': False}], '18.5')

    def test_boot_wait_is_bounded_and_timeout_propagates(self):
        with patch('scripts.boot_ios_simulator.subprocess.run', side_effect=subprocess.TimeoutExpired('xcrun', 240)) as process:
            with self.assertRaises(subprocess.TimeoutExpired):
                run('xcrun', 'simctl', 'bootstatus', 'device', '-b', timeout=240)
            self.assertEqual(process.call_args.kwargs['timeout'], 240)
            self.assertTrue(process.call_args.kwargs['check'])
