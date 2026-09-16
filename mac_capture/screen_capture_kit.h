#ifndef SCREEN_CAPTURE_KIT
#define SCREEN_CAPTURE_KIT

#import <ScreenCaptureKit/ScreenCaptureKit.h>

@interface StreamOutputHandler : NSObject <SCStreamOutput>
- (void) stream:(SCStream *) stream didOutputSampleBuffer:(CMSampleBufferRef) sampleBuffer ofType:(SCStreamOutputType) type;
@end

void start_capture();

// void capture_frame();
// void end_capture();

// void* create_sc_stream_configuration();

// void get_shareable_content_excluding_desktop_windows();

#endif // SCREEN_CAPTURE_KIT