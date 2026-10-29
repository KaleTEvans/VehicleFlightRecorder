"""Offline session validation and deterministic JSON summaries."""

import argparse
import json
import sys

from .session import IncompleteSessionError, ValidationError, load_session


def main(argv=None):
    parser = argparse.ArgumentParser(prog="python -m vehicle_flight_recorder")
    commands = parser.add_subparsers(dest="command", required=True)
    for name in ("validate", "summarize", "incident", "validate-connection"):
        command = commands.add_parser(name)
        command.add_argument("session_directory")
        if name == "incident":
            command.add_argument("incident_id")
        command.add_argument("--allow-incomplete", action="store_true", help="inspect well-formed unfinalized logs without repairing them")
    args = parser.parse_args(argv)
    try:
        if args.command == "validate-connection":
            from .connection import validate_connection
            summary = validate_connection(args.session_directory, allow_incomplete=args.allow_incomplete)
        elif args.command == "incident":
            from .incidents import reconstruct
            summary = reconstruct(args.session_directory, args.incident_id, allow_incomplete=args.allow_incomplete)
        else:
            summary = load_session(args.session_directory, allow_incomplete=args.allow_incomplete)
        if args.command == "validate":
            summary = {key: summary[key] for key in ("status", "session_id", "state", "observation_count", "event_count", "incident_count")}
        print(json.dumps(summary, sort_keys=True, indent=2, allow_nan=False))
        return 0
    except IncompleteSessionError as error:
        print(f"incomplete session: {error}", file=sys.stderr)
        return 2
    except (ValidationError, OSError) as error:
        print(f"invalid session: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
