import XCTest

final class EvidenceNavigationTests: XCTestCase {
    func testBothTriggerButtonsRevealTimeline() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["--ui-test-evidence"]
        app.launch()
        let voltage = app.buttons["trigger-evidence-voltage_excursion"]
        XCTAssertTrue(voltage.waitForExistence(timeout: 30), app.debugDescription)
        let heading = app.staticTexts["canonical-evidence-heading"]
        let scroll = app.scrollViews.firstMatch
        XCTAssertTrue(heading.exists)

        for kind in ["voltage_excursion", "module_timeout", "voltage_excursion"] {
            // Return to the capture/trigger area between taps, including repeat taps.
            for _ in 0..<3 {
                if voltage.isHittable { break }
                scroll.swipeDown()
            }
            let button = app.buttons["trigger-evidence-\(kind)"]
            if !button.isHittable { scroll.swipeUp() }
            XCTAssertTrue(button.isHittable)
            let before = heading.frame.minY
            button.tap()
            let revealed = NSPredicate { _, _ in
                let frame = heading.frame
                return heading.isHittable && frame.minY >= scroll.frame.minY &&
                    frame.maxY < scroll.frame.minY + scroll.frame.height * 0.6 &&
                    frame.minY < before - 50
            }
            expectation(for: revealed, evaluatedWith: heading)
            waitForExpectations(timeout: 8)
            let screenshot = XCTAttachment(screenshot: app.screenshot())
            screenshot.name = "Timeline revealed after \(kind)"
            screenshot.lifetime = .keepAlways
            add(screenshot)
        }
    }
    func testDashboardSurvivesBackgroundAndLabelsInterruptedRelaunch() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launch()
        let start = app.buttons["Start mock drive"]
        XCTAssertTrue(start.waitForExistence(timeout: 15))
        start.tap()
        let stop = app.buttons["Stop and save drive"]
        XCTAssertTrue(stop.waitForExistence(timeout: 10))
        let elapsed = app.staticTexts["drive-elapsed"]
        expectation(for: NSPredicate { _, _ in elapsed.label != "00:00" }, evaluatedWith: elapsed)
        waitForExpectations(timeout: 10)
        XCUIDevice.shared.press(.home)
        app.activate()
        XCTAssertTrue(stop.waitForExistence(timeout: 10))
        app.terminate()
        app.launch()
        app.tabBars.buttons["Sessions"].tap()
        let interrupted = app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", "Interrupted / incomplete")).firstMatch
        XCTAssertTrue(interrupted.waitForExistence(timeout: 10))
    }

    func testZeroSamplePhysicalHistoryExportOpensOnFirstTapAndRepeats() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["--ui-test-history-export"]
        app.launch()
        let history = app.buttons["Saved sessions, incidents and export"]
        XCTAssertTrue(history.waitForExistence(timeout: 30), app.debugDescription)
        history.tap()
        let rows = app.buttons.matching(NSPredicate(format: "identifier BEGINSWITH %@", "session-row-live-"))
        expectation(for: NSPredicate { _, _ in rows.count == 2 }, evaluatedWith: app)
        waitForExpectations(timeout: 15)
        let identifiers = rows.allElementsBoundByIndex.map(\.identifier)
        for (index, identifier) in identifiers.enumerated() {
            let row = app.buttons[identifier]
            let workload = row.staticTexts["session-workload-summary"].label
            XCTAssertTrue(["RPM only", "Coolant only"].contains(workload))
            row.tap()
            let displayedWorkload = app.staticTexts["session-workload-value"]
            for _ in 0..<4 {
                if displayedWorkload.isHittable { break }
                app.swipeUp()
            }
            XCTAssertTrue(displayedWorkload.waitForExistence(timeout: 10))
            // LabeledContent exposes a combined label/value to accessibility.
            XCTAssertEqual(displayedWorkload.label, "Workload, \(workload)")
            let requested = workload == "RPM only" ? "Engine RPM" : "Coolant temperature"
            XCTAssertTrue(app.staticTexts["Requested readings, \(requested)"].exists)
            let noReadings = app.staticTexts["session-no-readings"]
            for _ in 0..<3 {
                if noReadings.isHittable { break }
                app.swipeUp()
            }
            XCTAssertTrue(app.staticTexts["Operating state, Engine running / stationary"].exists)
            XCTAssertTrue(app.staticTexts["Origin, Simulated native test fixture"].exists)
            XCTAssertTrue(noReadings.isHittable)
            let setup = XCTAttachment(screenshot: app.screenshot())
            setup.name = "Saved \(workload) recording setup"
            setup.lifetime = .keepAlways
            add(setup)
            let export = app.buttons["export-session-zip"]
            // List materializes offscreen rows only after scrolling. Setup now
            // leads the detail page, so reveal Export before waiting for its row.
            for _ in 0..<6 {
                if export.exists && export.isHittable { break }
                app.collectionViews.firstMatch.swipeUp()
            }
            XCTAssertTrue(export.waitForExistence(timeout: 10), app.debugDescription)
            XCTAssertTrue(export.isHittable, app.debugDescription)
            for attempt in 0..<(index == 0 ? 2 : 1) {
                export.tap() // One tap; no retry loop can hide a presentation failure.
                // UIActivityViewController presents a remote system hierarchy;
                // verify its real controls and the correct file, not a host-view ID.
                let sheet = app.otherElements["ActivityListView"]
                XCTAssertTrue(sheet.waitForExistence(timeout: 15), app.debugDescription)
                let sessionID = String(identifier.dropFirst("session-row-".count))
                let title = app.navigationBars["UIActivityContentView"].otherElements.matching(NSPredicate(format: "label == %@", sessionID)).firstMatch
                XCTAssertTrue(title.waitForExistence(timeout: 30), app.debugDescription)
                let save = app.cells.matching(NSPredicate(format: "label == %@", "Save to Files")).firstMatch
                XCTAssertTrue(save.waitForExistence(timeout: 15), app.debugDescription)
                XCTAssertTrue(save.isHittable, app.debugDescription)
                XCTAssertFalse(app.staticTexts["session-export-error"].exists)
                let screenshot = XCTAttachment(screenshot: app.screenshot())
                screenshot.name = "Zero-sample export \(index)-\(attempt)"
                screenshot.lifetime = .keepAlways
                add(screenshot)
                let close = app.buttons["header.closeButton"].firstMatch
                XCTAssertTrue(close.isHittable, app.debugDescription)
                close.tap()
                expectation(for: NSPredicate { _, _ in !sheet.exists && export.isHittable && export.isEnabled }, evaluatedWith: app)
                waitForExpectations(timeout: 10)
            }
            app.navigationBars.buttons["Sessions"].tap()
            XCTAssertTrue(app.buttons[identifier].waitForExistence(timeout: 10))
        }
    }

    func testDiscoveryDiagnosticsShowNoDataAndLaterSupport() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["--ui-test-discovery"]
        app.launch()
        let outcome = app.staticTexts["discovery-outcome"]
        XCTAssertTrue(outcome.waitForExistence(timeout: 30), app.debugDescription)
        XCTAssertEqual(outcome.label, "ECU discovery: No data returned")
        XCTAssertEqual(app.staticTexts["discovery-setup"].label, "Adapter setup: 21/21")
        XCTAssertTrue(app.staticTexts["discovery-unknown"].exists)
        XCTAssertTrue(app.staticTexts["discovery-retry"].exists)
        XCTAssertTrue(app.staticTexts["discovery-can-status"].label.contains("T:00 R:00"))
        let screenshot = XCTAttachment(screenshot: app.screenshot())
        screenshot.name = "Diagnostic no-data status (scripted)"
        screenshot.lifetime = .keepAlways
        add(screenshot)
        let supported = app.buttons["Load supported reply fixture"]
        for _ in 0..<4 {
            if supported.exists && supported.isHittable { break }
            app.collectionViews.firstMatch.swipeUp()
        }
        XCTAssertTrue(supported.isHittable, app.debugDescription)
        supported.tap()
        for _ in 0..<4 {
            if outcome.isHittable { break }
            app.collectionViews.firstMatch.swipeDown()
        }
        expectation(for: NSPredicate { _, _ in outcome.label == "ECU discovery: Response received" }, evaluatedWith: app)
        waitForExpectations(timeout: 30)
        XCTAssertFalse(app.staticTexts["diagnostic-fixture-error"].exists)
        XCTAssertFalse(app.staticTexts["discovery-unknown"].exists)
    }

    func testAdapterHandoffDisplaysUnverifiedAndUnknownSettings() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["--ui-test-handoff"]
        app.launch()
        let count = app.staticTexts["handoff-count"]
        XCTAssertTrue(count.waitForExistence(timeout: 30), app.debugDescription)
        XCTAssertEqual(count.label, "Adapter settings captured: 3/3")
        XCTAssertEqual(app.staticTexts["handoff-protocol"].label, "Protocol number: 34")
        XCTAssertTrue(app.staticTexts["handoff-caveat"].exists)
        XCTAssertFalse(app.staticTexts["discovery-outcome"].exists)
        let shot = XCTAttachment(screenshot: app.screenshot())
        shot.name = "Unverified adapter handoff snapshot (scripted)"
        shot.lifetime = .keepAlways
        add(shot)
        let unknown = app.buttons["Show unknown settings fixture"]
        for _ in 0..<4 {
            if unknown.exists && unknown.isHittable { break }
            app.collectionViews.firstMatch.swipeUp()
        }
        XCTAssertTrue(unknown.isHittable, app.debugDescription)
        unknown.tap()
        for _ in 0..<4 {
            if app.staticTexts["handoff-protocol"].isHittable { break }
            app.collectionViews.firstMatch.swipeDown()
        }
        expectation(for: NSPredicate { _, _ in app.staticTexts["handoff-protocol"].label == "Protocol number: AUTO" }, evaluatedWith: app)
        waitForExpectations(timeout: 30)
        XCTAssertEqual(app.staticTexts["handoff-baud"].label, "Baud rate reply: ?")
        XCTAssertFalse(app.staticTexts["handoff-fixture-error"].exists)
    }

    func testCAN29DiagnosticProfileAndDiscoveryRecovery() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["--ui-test-can29"]
        app.launch()
        let profile = app.staticTexts["discovery-can29-profile"]
        XCTAssertTrue(profile.waitForExistence(timeout: 30), app.debugDescription)
        XCTAssertEqual(profile.label, "Profile: CAN 29-bit / 500 kbps")
        let outcome = app.staticTexts["discovery-outcome"]
        XCTAssertEqual(outcome.label, "ECU discovery: No data returned")
        XCTAssertEqual(app.staticTexts["discovery-setup"].label, "Adapter setup: 21/21")
        let supported = app.buttons["Load supported reply fixture"]
        for _ in 0..<4 {
            if supported.exists && supported.isHittable { break }
            app.collectionViews.firstMatch.swipeUp()
        }
        XCTAssertTrue(supported.isHittable, app.debugDescription)
        supported.tap()
        for _ in 0..<4 {
            if outcome.isHittable { break }
            app.collectionViews.firstMatch.swipeDown()
        }
        expectation(for: NSPredicate { _, _ in outcome.label == "ECU discovery: Response received" }, evaluatedWith: app)
        waitForExpectations(timeout: 30)
        XCTAssertFalse(app.staticTexts["diagnostic-fixture-error"].exists)
        let screenshot = XCTAttachment(screenshot: app.screenshot())
        screenshot.name = "CAN29 profile and supported discovery (scripted)"
        screenshot.lifetime = .keepAlways
        add(screenshot)
    }

}
