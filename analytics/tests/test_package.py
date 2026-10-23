"""Check the installed distribution, without injecting src into sys.path."""

from importlib.metadata import version

import vehicle_flight_recorder


def test_installed_package_version():
    assert version("vehicle-flight-recorder") == vehicle_flight_recorder.__version__
    assert vehicle_flight_recorder.__version__ == "0.1.0"
