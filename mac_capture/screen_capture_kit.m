#include "screen_capture_kit.h"

#include "_cgo_export.h"

#import <ScreenCaptureKit/ScreenCaptureKit.h>
#import <IOSurface/IOSurfaceRef.h>

#include <time.h>

// store the width and height of previous frame and malloc once and keep reusing buffer if size doesn't change
@interface StreamOutputHandler : NSObject <SCStreamOutput, SCStreamDelegate> {
    int prev_width;
    int prev_height;
    pixel_t* pixel_data;
    int frames_done;
    uint64_t last_frame_ns; // when the previous screen frame callback started, 0 before the first one
}
- (void) stream:(SCStream *) stream didOutputSampleBuffer:(CMSampleBufferRef) sampleBuffer ofType:(SCStreamOutputType) type;
- (void) stream:(SCStream *) stream didStopWithError:(NSError *) error;
@end

@implementation StreamOutputHandler
-(StreamOutputHandler*) init {
    self = [super init];
    if (self) {
        prev_width = 0;
        prev_height = 0;
        pixel_data = nil;
        frames_done = 0;
        last_frame_ns = 0;
    }
    return self;
}

// TODO: maybe this should just return the CVPixelBufferRef (or even the CMSampleBufferRef) to Go and Go can do the rest of the processing work
- (void) stream:(SCStream *) stream didOutputSampleBuffer:(CMSampleBufferRef) sampleBuffer ofType:(SCStreamOutputType) type {

    uint64_t frame_start_ns = clock_gettime_nsec_np(CLOCK_UPTIME_RAW);
    if (type != SCStreamOutputTypeScreen) return;
    if (!CMSampleBufferDataIsReady(sampleBuffer)) return;

    CVPixelBufferRef pixel_buffer = CMSampleBufferGetImageBuffer(sampleBuffer);
    IOSurfaceRef io_surface = CVPixelBufferGetIOSurface(pixel_buffer);
    size_t io_surface_width = IOSurfaceGetWidth(io_surface);
    size_t io_surface_height = IOSurfaceGetHeight(io_surface);
    size_t io_surface_planes = IOSurfaceGetPlaneCount(io_surface);

    const bool is_empty = io_surface_width == 0 || io_surface_height == 0;

    last_frame_ns = frame_start_ns;

    if (prev_width != io_surface_width || prev_height != io_surface_height) {
        if (pixel_data) {
            free(pixel_data);
        }

        pixel_data = (pixel_t*)malloc(io_surface_width * io_surface_height * sizeof(pixel_t));
    }

    memset(pixel_data, 0, io_surface_width * io_surface_height * sizeof(pixel_t));

    size_t bytes_per_row = IOSurfaceGetBytesPerRow(io_surface);

    if (!is_empty) {
        IOSurfaceLock(io_surface, kIOSurfaceLockReadOnly, nil);
        uint8_t* buffer = (uint8_t*)IOSurfaceGetBaseAddress(io_surface);

        for (int row = 0; row < io_surface_height; row++) {
            uint8_t* row_buffer_start = &buffer[row * bytes_per_row];
            for (int col = 0; col < io_surface_width; col++) {
                uint8_t* io_pixel = &row_buffer_start[col*4];
                uint8_t b = io_pixel[0];
                uint8_t g = io_pixel[1];
                uint8_t r = io_pixel[2];
                uint8_t a = io_pixel[3];

                pixel_t* pixel = &pixel_data[(row*io_surface_width) + col];
                pixel->r = r;
                pixel->g = g;
                pixel->b = b;
                pixel->a = a;
            }
        }

        IOSurfaceUnlock(io_surface, kIOSurfaceLockReadOnly, nil);
    }

    frame_t frame;
    frame.width = (int)io_surface_width;
    frame.height = (int)io_surface_height;
    frame.num_planes = (int)io_surface_planes;
    frame.pixel_data = is_empty ? nil : pixel_data;

    GoTransformFrame(frame);

    prev_width = io_surface_width;
    prev_height = io_surface_height;

    frames_done++;
}

- (void) stream:(SCStream *) stream didStopWithError:(NSError *) error {
    printf("[objc] stream stopped with error: domain=%s code=%ld\n",
        error.domain.UTF8String, (long)error.code);
    printf("[objc]   description: %s\n", error.localizedDescription.UTF8String);
    if (error.localizedFailureReason) {
        printf("[objc]   reason: %s\n", error.localizedFailureReason.UTF8String);
    }
    if (error.localizedRecoverySuggestion) {
        printf("[objc]   suggestion: %s\n", error.localizedRecoverySuggestion.UTF8String);
    }
    NSError* underlying = error.userInfo[NSUnderlyingErrorKey];
    if (underlying) {
        printf("[objc]   underlying: domain=%s code=%ld %s\n",
            underlying.domain.UTF8String, (long)underlying.code, underlying.localizedDescription.UTF8String);
    }
    fflush(stdout);
}
@end

typedef struct capture_metadata_t {
    SCStreamConfiguration* config;
    StreamOutputHandler* output_handler;
    SCDisplay* display;
    SCContentFilter* content_filter;
    SCStream* stream;
} capture_metadata_t; 
static capture_metadata_t capture_metadata;

// NOTE: Is blocking main thread right now, maybe can move this to separate thread moving forward?
// or somehow make it so that main thread has some sort of callback with Go stack
void start_capture() {
    if (capture_metadata.stream) return;

    dispatch_semaphore_t capture_sem = dispatch_semaphore_create(0);
    
    [SCShareableContent getShareableContentExcludingDesktopWindows:false onScreenWindowsOnly:true completionHandler:^(SCShareableContent* shareable_content, NSError *error){
        // TODO: need to add better error handling for this whole block

        // just capture the first display
        if (shareable_content.displays.count <= 0) {
            dispatch_semaphore_signal(capture_sem);
            return;
        }

        SCDisplay* main_display = shareable_content.displays[0];
        capture_metadata.display = main_display;

        SCContentFilter* content_filter = [[SCContentFilter alloc] initWithDisplay:main_display excludingWindows:@[]];
        capture_metadata.content_filter = content_filter;

        SCStreamConfiguration* stream_config = [[SCStreamConfiguration alloc] init];
        stream_config.width = main_display.width;
        stream_config.height = main_display.height;
        stream_config.pixelFormat = kCVPixelFormatType_32BGRA;
        stream_config.queueDepth = 2;

        CMTime time;
        time.value = 0;
        time.timescale = 60;
        stream_config.minimumFrameInterval = time;

        capture_metadata.config = stream_config;

        StreamOutputHandler* stream_output_handler = [[StreamOutputHandler alloc] init];
        capture_metadata.output_handler = stream_output_handler;

        SCStream* stream = [[SCStream alloc] initWithFilter:content_filter configuration:stream_config delegate:stream_output_handler];
        capture_metadata.stream = stream;

        [stream addStreamOutput:stream_output_handler type:SCStreamOutputTypeScreen sampleHandlerQueue:nil error:nil];
        [stream startCaptureWithCompletionHandler:^(NSError* error) {
            if (error != nil) {
                printf("[objc] failed to start capture: %s\n", error.localizedDescription.UTF8String);
            }
            dispatch_semaphore_signal(capture_sem);
        }];
    }];

    dispatch_semaphore_wait(capture_sem, DISPATCH_TIME_FOREVER);
}

void stop_capture() {
    if (!capture_metadata.stream) return;

    dispatch_semaphore_t capture_sem = dispatch_semaphore_create(0);
    [capture_metadata.stream stopCaptureWithCompletionHandler:^(NSError* error) {
        dispatch_semaphore_signal(capture_sem);
    }];
    dispatch_semaphore_wait(capture_sem, DISPATCH_TIME_FOREVER);

    capture_metadata.stream = nil;
    capture_metadata.output_handler = nil;
    capture_metadata.config = nil;
    capture_metadata.content_filter = nil;
    capture_metadata.display = nil;
}