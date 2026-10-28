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
}
