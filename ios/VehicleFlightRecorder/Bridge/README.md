RecorderService is the sole native actor owner of a real MobileController.
It verifies bridge contract 1 before storage initialization, translates Go NSError
prefixes into BridgeFailure categories, and checks cancellation before calls.
Session-bound advance and snapshot run atomically within one actor operation.
History pages are bounded and explicit; close is terminal and preserves unfinished
metadata. Swift presentation does not invent measurements or apply pack rules.

See [bridge contract](../../../docs/BRIDGE_CONTRACT.md),
[live timing design](../../../docs/LIVE_TIMING.md) and
[Milestone 4 evidence](../../../docs/MILESTONE_4.md).
