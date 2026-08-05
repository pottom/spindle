//go:build darwin

#import <AppKit/AppKit.h>
#import <CoreGraphics/CoreGraphics.h>

// The hardware play, next and previous keys, as the media key event reports
// them. They are the numbers Apple has used since the first keyboard that had
// them, and they are not in any public header worth including for three
// constants.
enum {
    mediaKeyPlay = 16,
    mediaKeyNext = 19,
    mediaKeyPrev = 20,
};

// spindleMediaKey is the Go side. It answers whether the press was used, which
// is how the key is either taken or passed on to whatever else was listening.
extern int spindleMediaKey(int key);

static CFMachPortRef tap = NULL;

// mediaKeyTap reads the system-defined events the media keys arrive as.
//
// The event carries its meaning in a field that only NSEvent knows how to take
// apart, which is why AppKit is here at all: no window is opened and no
// application is started, the class is used as the decoder it is.
static CGEventRef mediaKeyTap(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *context) {
    // A tap the system has switched off — because it was too slow, or because
    // the screen was locked — is switched back on rather than left dead.
    if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
        if (tap != NULL) {
            CGEventTapEnable(tap, true);
        }
        return event;
    }
    if (type != NX_SYSDEFINED) {
        return event;
    }

    NSEvent *ns = [NSEvent eventWithCGEvent:event];
    if (ns == nil || [ns subtype] != NSEventSubtypeScreenChanged) {
        // Subtype 8 is what the media keys use, and AppKit calls it by a name
        // that has nothing to do with what it means here.
        return event;
    }

    int key = ([ns data1] & 0xFFFF0000) >> 16;
    int flags = [ns data1] & 0x0000FFFF;
    int down = ((flags & 0xFF00) >> 8) == 0x0A;
    int repeated = flags & 0x1;

    if (key != mediaKeyPlay && key != mediaKeyNext && key != mediaKeyPrev) {
        return event;
    }
    // Acted on when the key goes down, once: the release is the same event
    // again, and holding a media key is not a gesture anybody means.
    if (!down || repeated) {
        return event;
    }

    if (spindleMediaKey(key)) {
        return NULL; // taken, so nothing else hears it
    }
    return event;
}

// mediaKeysAllowed asks the system whether this process may listen to keys at
// all. Worth asking rather than assuming: without the permission the tap is
// still created and simply never hears anything, which is indistinguishable
// from a keyboard nobody is pressing.
int mediaKeysAllowed(void) { return CGPreflightListenEventAccess() ? 1 : 0; }

// askForMediaKeys puts the system's own request on screen, once. Everything
// after that is in the user's hands and in System Settings.
void askForMediaKeys(void) { CGRequestListenEventAccess(); }

// startMediaKeys begins listening. It answers 0 when the system refuses, which
// it does until the terminal running spindle is allowed to monitor input.
int startMediaKeys(void) {
    CGEventMask mask = CGEventMaskBit(NX_SYSDEFINED);
    tap = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap,
                           kCGEventTapOptionDefault, mask, mediaKeyTap, NULL);
    if (tap == NULL) {
        return 0;
    }

    CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, tap, 0);
    CFRunLoopAddSource(CFRunLoopGetCurrent(), source, kCFRunLoopCommonModes);
    CGEventTapEnable(tap, true);
    return 1;
}

// runMediaKeys hands this thread to the run loop the tap needs. It does not
// return.
void runMediaKeys(void) { CFRunLoopRun(); }
