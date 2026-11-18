//go:build cgo && !nogui && darwin

package main

/*
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static void keimlingEditMenu(void) {
	if (NSApp == nil) {
		return;
	}
	NSMenu *bar = [[NSMenu alloc] init];
	NSMenuItem *appItem = [[NSMenuItem alloc] init];
	NSMenu *app = [[NSMenu alloc] initWithTitle:@"Keimling"];
	[app addItemWithTitle:@"Quit" action:@selector(terminate:) keyEquivalent:@"q"];
	[appItem setSubmenu:app];
	[bar addItem:appItem];

	NSMenuItem *editItem = [[NSMenuItem alloc] init];
	NSMenu *edit = [[NSMenu alloc] initWithTitle:@"Edit"];
	[edit addItemWithTitle:@"Undo" action:@selector(undo:) keyEquivalent:@"z"];
	[edit addItemWithTitle:@"Redo" action:@selector(redo:) keyEquivalent:@"Z"];
	[edit addItem:[NSMenuItem separatorItem]];
	[edit addItemWithTitle:@"Cut" action:@selector(cut:) keyEquivalent:@"x"];
	[edit addItemWithTitle:@"Copy" action:@selector(copy:) keyEquivalent:@"c"];
	[edit addItemWithTitle:@"Paste" action:@selector(paste:) keyEquivalent:@"v"];
	[edit addItemWithTitle:@"Select All" action:@selector(selectAll:) keyEquivalent:@"a"];
	[editItem setSubmenu:edit];
	[bar addItem:editItem];
	[NSApp setMainMenu:bar];
}
*/
import "C"

func prepareDesktopApp() { C.keimlingEditMenu() }
