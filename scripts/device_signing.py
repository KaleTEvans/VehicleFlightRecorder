"""Validate a supplied App Store distribution profile and generate export options."""
import datetime
import plistlib
import sys
from pathlib import Path

BUNDLE = 'com.kaleevans.vehicleflightrecorder'


def export_options(profile):
    team = profile['TeamIdentifier'][0]
    entitlements = profile['Entitlements']
    expiry = profile['ExpirationDate'].replace(tzinfo=datetime.timezone.utc)
    if expiry <= datetime.datetime.now(datetime.timezone.utc):
        raise ValueError('Provisioning profile has expired')
    if entitlements.get('application-identifier') != team + '.' + BUNDLE:
        raise ValueError('Profile does not match the approved app identity')
    if entitlements.get('get-task-allow') or profile.get('ProvisionedDevices') or profile.get('ProvisionsAllDevices'):
        raise ValueError('Expected an App Store distribution profile')
    return {'method': 'app-store-connect', 'signingStyle': 'manual', 'teamID': team,
            'signingCertificate': 'Apple Distribution', 'provisioningProfiles': {BUNDLE: profile['UUID']},
            'uploadSymbols': True, 'manageAppVersionAndBuildNumber': False}


def main():
    source, output, environment = map(Path, sys.argv[1:])
    with source.open('rb') as file:
        profile = plistlib.load(file)
    options = export_options(profile)
    with output.open('wb') as file:
        plistlib.dump(options, file)
    with environment.open('a') as file:
        file.write(f"SIGN_TEAM={options['teamID']}\nSIGN_PROFILE={profile['UUID']}\n")


if __name__ == '__main__':
    main()
