RecorderModel owns presentation state and explicit actions on the main actor; RecorderService serializes blocking Go calls off the UI actor. No view recomputation creates markers.
