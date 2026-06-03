#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>
#include <stdlib.h>
#include <string.h>

static NSStatusItem *statusItem = nil;

void initStatusBar(const char *initialTitle) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

        statusItem = [[[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength] retain];
        statusItem.button.title = initialTitle ? [NSString stringWithUTF8String:initialTitle] : @"";
        statusItem.button.toolTip = @"sock-emoji";

        NSMenu *menu = [[NSMenu alloc] init];
        [menu addItemWithTitle:@"Quit" action:@selector(terminate:) keyEquivalent:@"q"];
        statusItem.menu = menu;
    }
}

void setStatusBarTitle(const char *title) {
    char *titleCopy = title ? strdup(title) : NULL;

    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            if (statusItem == nil) {
                free(titleCopy);
                return;
            }

            NSString *nextTitle = titleCopy ? [NSString stringWithUTF8String:titleCopy] : @"";
            if (nextTitle == nil) {
                nextTitle = @"";
            }

            statusItem.button.title = nextTitle;
            free(titleCopy);
        }
    });
}

void stopApp(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            [NSApp terminate:nil];
        }
    });
}

void cleanupStatusBar(void) {
    @autoreleasepool {
        if (statusItem != nil) {
            [[NSStatusBar systemStatusBar] removeStatusItem:statusItem];
            [statusItem release];
            statusItem = nil;
        }
    }
}

void runApp(void) {
    @autoreleasepool {
        [NSApp run];
    }
}
