//go:build darwin && cgo

#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

static NSPrintOperation *createPrintOperation(WKWebView *webView) {
    NSPrintInfo *info = [[NSPrintInfo sharedPrintInfo] copy];
    info.orientation = NSPaperOrientationPortrait;
    info.horizontalPagination = NSPrintingPaginationModeFit;
    info.verticalPagination = NSPrintingPaginationModeAutomatic;
    info.horizontallyCentered = NO;
    info.verticallyCentered = NO;
    info.leftMargin = 0;
    info.rightMargin = 0;
    info.topMargin = 0;
    info.bottomMargin = 0;
    NSPrintOperation *operation = [webView printOperationWithPrintInfo:info];
    [info release];
    operation.view.frame = webView.bounds;
    return operation;
}

static WKWebView *findWebView(NSView *view) {
    if ([view isKindOfClass:[WKWebView class]]) {
        return (WKWebView *)view;
    }
    for (NSView *child in view.subviews) {
        WKWebView *webView = findWebView(child);
        if (webView != nil) {
            return webView;
        }
    }
    return nil;
}

typedef struct {
    dispatch_semaphore_t completed;
    int result;
} PrintRequest;

@interface TtyphoonPrintCompletion : NSObject
@property(nonatomic, assign) PrintRequest *request;
@end

@implementation TtyphoonPrintCompletion
- (void)printOperationDidRun:(NSPrintOperation *)operation success:(BOOL)success contextInfo:(void *)context {
    dispatch_semaphore_signal(self.request->completed);
    [self release];
}
@end

int ttyphoon_print_window(void) {
    if (NSApp == nil) {
        return 1;
    }
    if (@available(macOS 11.0, *)) {
        PrintRequest request = {dispatch_semaphore_create(0), 0};
        PrintRequest *pending = &request;
        void (^printOperation)(void) = ^{
            @autoreleasepool {
            TtyphoonPrintCompletion *completion = nil;
                @try {
                    NSWindow *window = NSApp.mainWindow ?: NSApp.keyWindow;
                    WKWebView *webView = findWebView(window.contentView);
                    if (webView == nil) {
                        pending->result = 1;
                        dispatch_semaphore_signal(pending->completed);
                        return;
                    }
                    NSPrintOperation *operation = createPrintOperation(webView);
                    if (operation == nil) {
                        pending->result = 3;
                        dispatch_semaphore_signal(pending->completed);
                        return;
                    }
                    operation.showsPrintPanel = YES;
                    operation.showsProgressPanel = YES;
                    completion = [TtyphoonPrintCompletion new];
                    completion.request = pending;
                    [operation runOperationModalForWindow:window delegate:completion
                        didRunSelector:@selector(printOperationDidRun:success:contextInfo:) contextInfo:NULL];
                } @catch (NSException *exception) {
                    NSLog(@"ttyphoon printing failed: %@", exception);
                    [completion release];
                    pending->result = 3;
                    dispatch_semaphore_signal(pending->completed);
                }
            }
        };
        if ([NSThread isMainThread]) {
            printOperation();
            while (dispatch_semaphore_wait(request.completed, DISPATCH_TIME_NOW) != 0) {
                CFRunLoopRunInMode(kCFRunLoopDefaultMode, 0.05, true);
            }
        } else {
            dispatch_async(dispatch_get_main_queue(), printOperation);
            dispatch_semaphore_wait(request.completed, DISPATCH_TIME_FOREVER);
        }
        dispatch_release(request.completed);
        return request.result;
    }
    return 2;
}