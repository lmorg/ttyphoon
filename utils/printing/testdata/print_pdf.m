#import <PDFKit/PDFKit.h>
#import "../printing_darwin.m"

@interface PrintFixture : NSObject <WKNavigationDelegate>
@property(nonatomic, retain) NSURL *outputURL;
@end

@implementation PrintFixture
- (void)webView:(WKWebView *)webView didFinishNavigation:(WKNavigation *)navigation {
    fprintf(stderr, "navigation complete\n");
    [self performSelector:@selector(savePDF:) withObject:webView afterDelay:0];
}

- (void)savePDF:(WKWebView *)webView {
    NSPrintOperation *operation = createPrintOperation(webView);
    fprintf(stderr, "print view frame: %s\n", NSStringFromRect(operation.view.frame).UTF8String);
    operation.showsPrintPanel = NO;
    operation.showsProgressPanel = NO;
    operation.printInfo.jobDisposition = NSPrintSaveJob;
    operation.printInfo.dictionary[NSPrintJobSavingURL] = self.outputURL;
    fprintf(stderr, "saving PDF\n");
    [operation runOperationModalForWindow:webView.window delegate:self
        didRunSelector:@selector(printOperationDidRun:success:contextInfo:) contextInfo:NULL];
}

- (void)printOperationDidRun:(NSPrintOperation *)operation success:(BOOL)completed contextInfo:(void *)context {
    PDFDocument *document = [[PDFDocument alloc] initWithURL:self.outputURL];
    NSString *text = document.string ?: @"";
    fprintf(stdout, "completed=%d pages=%lu characters=%lu\n", completed,
        (unsigned long)document.pageCount, (unsigned long)text.length);
    BOOL containsStart = [text containsString:@"PRINT START SENTINEL"];
    BOOL containsEnd = [text containsString:@"PRINT END SENTINEL"];
    BOOL completeDocument = document.pageCount > 1;
    for (int index = 0; index < 120; index++) {
        completeDocument &= [text containsString:[NSString stringWithFormat:@"Paragraph %d of", index]];
    }
    BOOL excludesApplication = ![text containsString:@"APPLICATION SIDEBAR"];
    NSRect page = [[document pageAtIndex:0] boundsForBox:kPDFDisplayBoxMediaBox];
    BOOL portrait = page.size.width < page.size.height;
    fprintf(stdout, "start=%d end=%d complete_document=%d excludes_application=%d portrait=%d\n",
        containsStart, containsEnd, completeDocument, excludesApplication, portrait);
    fflush(stdout);
    exit(completed && containsStart && containsEnd && completeDocument && excludesApplication && portrait ? 0 : 1);
}
@end

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        if (argc != 3) return 2;
        dispatch_after(dispatch_time(DISPATCH_TIME_NOW, 30 * NSEC_PER_SEC),
            dispatch_get_global_queue(QOS_CLASS_UTILITY, 0), ^{
                fprintf(stderr, "native PDF fixture timed out\n");
                _Exit(2);
            });
        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyProhibited];
        NSWindow *window = [[NSWindow alloc] initWithContentRect:NSMakeRect(0, 0, 1000, 700)
            styleMask:NSWindowStyleMaskBorderless backing:NSBackingStoreBuffered defer:NO];
        WKWebView *webView = [[WKWebView alloc] initWithFrame:window.contentView.bounds];
        [window.contentView addSubview:webView];
        PrintFixture *fixture = [PrintFixture new];
        fixture.outputURL = [NSURL fileURLWithPath:[NSString stringWithUTF8String:argv[2]]];
        webView.navigationDelegate = fixture;
        NSString *css = [NSString stringWithContentsOfFile:[NSString stringWithUTF8String:argv[1]]
            encoding:NSUTF8StringEncoding error:nil];
        if (css == nil) return 2;
        NSMutableString *html = [NSMutableString stringWithFormat:
            @"<!doctype html><html class='notes-print-ready'><head><style>%@</style></head>"
            "<body><aside>APPLICATION SIDEBAR</aside><article id='notes-print-root'>"
            "<h1>PRINT START SENTINEL</h1>", css];
        for (int index = 0; index < 120; index++) {
            [html appendFormat:@"<p>Paragraph %d of the complete printable document.</p>", index];
        }
        [html appendString:@"<p>PRINT END SENTINEL</p></article></body></html>"];
        [webView loadHTMLString:html baseURL:nil];
        fprintf(stderr, "running hidden webview\n");
        [NSApp run];
    }
    return 2;
}