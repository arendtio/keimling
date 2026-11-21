# User stories: Camera share

## US-79 — Let the assistant see through my camera

**As** a user, **I want** to turn my camera on beside the chat, or let the assistant take a still, **so that** it can answer about what is in front of me without a screenshot every time.

**Acceptance:** traces REQ-CAM-1, REQ-CAM-2, REQ-CAM-3, REQ-CAM-4, REQ-CAM-5, and REQ-CAM-6. The camera switch sits beside the chat, next to the screen switch. The camera picture runs in the Now view with a status line and Stop, and it keeps running when a screen is shared at the same time. Only the newest camera frame reaches the model. A `camera_snapshot` call with the camera off asks the user to turn it on. The stream carries no sound, and Stop closes the camera and drops the frame.
