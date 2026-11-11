//go:build darwin && cgo

// Still frames go through ScreenCaptureKit. The macOS 15 SDK marks
// CGDisplayCreateImage and CGWindowListCreateImage unavailable.

#import <ScreenCaptureKit/ScreenCaptureKit.h>
#include <os/lock.h>
#include <stdlib.h>

typedef struct {
	os_unfair_lock lock;
	int alive;
	CGImageRef image;
	dispatch_semaphore_t sem;
} screenShotSlot;

static void screenShotDeliver(screenShotSlot *slot, CGImageRef image) {
	os_unfair_lock_lock(&slot->lock);
	if (slot->alive && image) slot->image = CGImageRetain(image);
	dispatch_semaphore_t sem = slot->sem;
	os_unfair_lock_unlock(&slot->lock);
	if (sem) dispatch_semaphore_signal(sem);
}

static void screenShotFree(screenShotSlot *slot) {
	if (!slot) return;
	if (slot->image) CFRelease(slot->image);
#if OS_OBJECT_USE_OBJC
	if (slot->sem) [(id)slot->sem release];
#else
	if (slot->sem) dispatch_release(slot->sem);
#endif
	free(slot);
}

static CGImageRef screenCaptureImage14(int display, uint32_t targetID) API_AVAILABLE(macos(14.0)) {
	screenShotSlot *slot = (screenShotSlot *)calloc(1, sizeof *slot);
	if (!slot) return NULL;
	slot->alive = 1;
	slot->sem = dispatch_semaphore_create(0);
	if (!slot->sem) {
		free(slot);
		return NULL;
	}

	[SCShareableContent getShareableContentWithCompletionHandler:^(SCShareableContent *content, NSError *error) {
		(void)error;
		@autoreleasepool {
			SCContentFilter *filter = nil;
			if (content && display) {
				for (SCDisplay *item in content.displays) {
					if (item.displayID == targetID) {
						filter = [[SCContentFilter alloc] initWithDisplay:item excludingWindows:@[]];
						break;
					}
				}
			} else if (content) {
				for (SCWindow *item in content.windows) {
					if (item.windowID == targetID) {
						filter = [[SCContentFilter alloc] initWithDesktopIndependentWindow:item];
						break;
					}
				}
			}
			if (!filter) {
				screenShotDeliver(slot, NULL);
				return;
			}
			CGFloat scale = filter.pointPixelScale;
			if (scale < 1) scale = 1;
			size_t pw = (size_t)(filter.contentRect.size.width * scale);
			size_t ph = (size_t)(filter.contentRect.size.height * scale);
			if (pw < 1 || ph < 1 || pw > 16000 || ph > 16000) {
				[filter release];
				screenShotDeliver(slot, NULL);
				return;
			}
			SCStreamConfiguration *cfg = [[SCStreamConfiguration alloc] init];
			cfg.width = pw;
			cfg.height = ph;
			cfg.showsCursor = NO;
			cfg.capturesAudio = NO;
			[SCScreenshotManager captureImageWithFilter:filter configuration:cfg completionHandler:^(CGImageRef image, NSError *captureError) {
				(void)captureError;
				screenShotDeliver(slot, image);
			}];
			[filter release];
			[cfg release];
		}
	}];

	int timedOut = dispatch_semaphore_wait(slot->sem, dispatch_time(DISPATCH_TIME_NOW, 3 * NSEC_PER_SEC)) != 0;
	CGImageRef image = NULL;
	os_unfair_lock_lock(&slot->lock);
	slot->alive = 0;
	image = slot->image;
	slot->image = NULL;
	os_unfair_lock_unlock(&slot->lock);
	if (timedOut) return image;
	screenShotFree(slot);
	return image;
}

CGImageRef screenCaptureImage(int display, uint32_t targetID) {
	if (@available(macOS 14.0, *)) return screenCaptureImage14(display, targetID);
	return NULL;
}
