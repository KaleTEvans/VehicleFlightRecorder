"""Reject empty/injected/mislabeled targeted XCTest runs before macOS builds."""
import unittest
from scripts.native_test_plan import make_plan, verify_results
from scripts.run_native_tests import command

UI = 'VehicleFlightRecorderUITests/EvidenceNavigationTests/testRestoredParkedValidationSetupRequiresFreshConfirmationOnlyWhenRearming'
UNIT = 'VehicleFlightRecorderTests/TransportDebugModelTests'
LOG = """Test Case '-[VehicleFlightRecorderTests.TransportDebugModelTests testPeriodicRefreshDoesNotDisableButtons]' passed (0.001 seconds).
Test Case '-[VehicleFlightRecorderUITests.EvidenceNavigationTests testRestoredParkedValidationSetupRequiresFreshConfirmationOnlyWhenRearming]' passed (9.0 seconds).
"""


class NativeTestPlanTests(unittest.TestCase):
    def test_full_acceptance_requires_both_test_targets(self):
        plan = make_plan('full')
        self.assertTrue(verify_results(plan, LOG)['full_acceptance'])
        with self.assertRaisesRegex(ValueError, 'No successful test'):
            verify_results(plan, LOG.splitlines()[0])

    def test_targeted_proves_every_requested_case(self):
        plan = make_plan('targeted', UNIT+', '+UI)
        result = verify_results(plan, LOG)
        self.assertFalse(result['full_acceptance'])
        self.assertEqual(len(result['passed_tests']), 2)
        with self.assertRaisesRegex(ValueError, 'No successful test'):
            verify_results(plan, LOG.splitlines()[0])

    def test_zero_tests_or_failure_text_cannot_qualify(self):
        plan = make_plan('targeted', UI)
        for text in ['', 'TEST SUCCEEDED', LOG.replace(' passed ', ' failed ')]:
            with self.assertRaises(ValueError): verify_results(plan, text)

    def test_unknown_or_injected_selector_fails_early(self):
        for value in ['-skip-testing:Everything', UI+';echo bad', UNIT+'/testDoesNotExist', 'VehicleFlightRecorderTests/DoesNotExist', UNIT+',', UNIT+','+UNIT]:
            with self.subTest(value=value), self.assertRaises(ValueError): make_plan('targeted', value)

    def test_no_accidental_full_filter_or_targeted_publish(self):
        for args in [('full', UI), ('targeted', ''), ('unknown', '')]:
            with self.assertRaises(ValueError): make_plan(*args)
        with self.assertRaises(ValueError): make_plan('targeted', UI, upload=True)

    def test_selectors_are_single_process_arguments(self):
        args = command('B2CA4CE2-6C5D-4677-AD8B-36DB16610C77', make_plan('targeted', UI))
        self.assertIn('-only-testing:'+UI, args)
        self.assertEqual(args[-1], 'test')
        with self.assertRaises(ValueError): command('not-a-simulator;bad', make_plan('full'))
