#include "screen_capture_kit.h"

#include "_cgo_export.h"

#import <ScreenCaptureKit/ScreenCaptureKit.h>
#import <IOSurface/IOSurfaceRef.h>

#include <time.h>
#include <os/lock.h>

// Per-section totals across all frames; averages are printed once in stop_capture.
// Frames arrive on ScreenCaptureKit's queue while stop_capture runs on the caller's thread.
typedef struct objc_frame_stats_t {
    uint64_t frames;
    uint64_t empty_frames;
    uint64_t surface_ns;
    uint64_t alloc_ns;
    uint64_t copy_ns;
    uint64_t go_ns;
    uint64_t total_ns;
} objc_frame_stats_t;
static objc_frame_stats_t objc_stats;
static os_unfair_lock objc_stats_lock = OS_UNFAIR_LOCK_INIT;

// store the width and height of previous frame and malloc once and keep reusing buffer if size doesn't change
@interface StreamOutputHandler : NSObject <SCStreamOutput, SCStreamDelegate> {
    int prev_width;
    int prev_height;
    pixel_t* pixel_data;
    int frames_done;
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
    }
    return self;
}

// TODO: maybe this should just return the CVPixelBufferRef (or even the CMSampleBufferRef) to Go and Go can do the rest of the processing work
- (void) stream:(SCStream *) stream didOutputSampleBuffer:(CMSampleBufferRef) sampleBuffer ofType:(SCStreamOutputType) type {

    if (frames_done > 60) return;

    // if (pixel_data != nil) return;
    uint64_t frame_start_ns = clock_gettime_nsec_np(CLOCK_UPTIME_RAW);
    if (type != SCStreamOutputTypeScreen) return;
    if (!CMSampleBufferDataIsReady(sampleBuffer)) return;

    static int logged_format = 0;

    CVPixelBufferRef pixel_buffer = CMSampleBufferGetImageBuffer(sampleBuffer);
    IOSurfaceRef io_surface = CVPixelBufferGetIOSurface(pixel_buffer);
    size_t io_surface_width = IOSurfaceGetWidth(io_surface);
    size_t io_surface_height = IOSurfaceGetHeight(io_surface);
    size_t io_surface_planes = IOSurfaceGetPlaneCount(io_surface);

    // printf("Buffer is %zu by %zu\n", io_surface_width, io_surface_height);

    const bool is_empty = io_surface_width == 0 || io_surface_height == 0;
    if (is_empty) {
        os_unfair_lock_lock(&objc_stats_lock);
        objc_stats.empty_frames++;
        os_unfair_lock_unlock(&objc_stats_lock);
    }

    OSType io_surface_pixel_format = IOSurfaceGetPixelFormat(io_surface);
    // Printing every frame is itself slow enough to skew the timings below,
    // so the one-off details only go out for the first frame.
    if (!logged_format) {
        // printf("Pixel format: '%c%c%c%c' (0x%08x), %zu planes\n",
        //    (char)((io_surface_pixel_format >> 24) & 0xFF),
        //    (char)((io_surface_pixel_format >> 16) & 0xFF),
        //    (char)((io_surface_pixel_format >> 8) & 0xFF),
        //    (char)(io_surface_pixel_format & 0xFF),
        //    (unsigned int)io_surface_pixel_format,
        //    io_surface_planes);
    }

    uint64_t t_surface_ns = clock_gettime_nsec_np(CLOCK_UPTIME_RAW);

    if (prev_width != io_surface_width || prev_height != io_surface_height) {
        if (pixel_data) {
            free(pixel_data);
        }

        pixel_data = (pixel_t*)malloc(io_surface_width * io_surface_height * sizeof(pixel_t));
    }

    memset(pixel_data, 0, io_surface_width * io_surface_height * sizeof(pixel_t));

    uint64_t t_alloc_ns = clock_gettime_nsec_np(CLOCK_UPTIME_RAW);

    size_t bytes_per_row = IOSurfaceGetBytesPerRow(io_surface);
    if (!logged_format) {
        // printf("%zu by %zu, bytes per row is %zu\n", io_surface_width, io_surface_height, bytes_per_row);
        logged_format = 1;
    }
    
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

                // printf("pixel is %u %u %u %u\n", r, g, b, a);
            }
        }

        IOSurfaceUnlock(io_surface, kIOSurfaceLockReadOnly, nil);
    }

    uint64_t t_copy_ns = clock_gettime_nsec_np(CLOCK_UPTIME_RAW);

    frame_t frame;
    frame.width = (int)io_surface_width;
    frame.height = (int)io_surface_height;
    frame.num_planes = (int)io_surface_planes;
    frame.pixel_data = is_empty ? nil : pixel_data;

    GoTransformFrame(frame);

    uint64_t frame_end_ns = clock_gettime_nsec_np(CLOCK_UPTIME_RAW);

    os_unfair_lock_lock(&objc_stats_lock);
    objc_stats.frames++;
    objc_stats.surface_ns += t_surface_ns - frame_start_ns;
    objc_stats.alloc_ns += t_alloc_ns - t_surface_ns;
    objc_stats.copy_ns += t_copy_ns - t_alloc_ns;
    objc_stats.go_ns += frame_end_ns - t_copy_ns;
    objc_stats.total_ns += frame_end_ns - frame_start_ns;
    os_unfair_lock_unlock(&objc_stats_lock);

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

        // printf("We have %i displays\n", (int)shareable_content.displays.count);

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
        stream_config.queueDepth = 3;

        CMTime time;
        time.value = 1;
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

// Prints a section's average time per frame and its share of the per-frame total.
static void print_time_row(const char* name, uint64_t sum_ns, uint64_t frames, uint64_t total_ns) {
    printf("    %-22s %6.2f ms  %3.0f%%\n", name, sum_ns / (double)frames / 1000000.0, 100.0 * sum_ns / (double)total_ns);
}

static void print_objc_averages() {
    os_unfair_lock_lock(&objc_stats_lock);
    objc_frame_stats_t stats = objc_stats;
    os_unfair_lock_unlock(&objc_stats_lock);

    if (stats.frames == 0) return;
    printf("\n[objc] frame callback: %.2f ms avg over %llu frames (%llu unchanged-screen callbacks skipped)\n",
        stats.total_ns / (double)stats.frames / 1000000.0, stats.frames, stats.empty_frames);
    print_time_row("copy pixels", stats.copy_ns, stats.frames, stats.total_ns);
    print_time_row("GoTransformFrame", stats.go_ns, stats.frames, stats.total_ns);
    print_time_row("other", stats.surface_ns + stats.alloc_ns, stats.frames, stats.total_ns);
    fflush(stdout);
}

void stop_capture() {
    if (!capture_metadata.stream) return;

    dispatch_semaphore_t capture_sem = dispatch_semaphore_create(0);
    [capture_metadata.stream stopCaptureWithCompletionHandler:^(NSError* error) {
        dispatch_semaphore_signal(capture_sem);
    }];
    dispatch_semaphore_wait(capture_sem, DISPATCH_TIME_FOREVER);
    // printf("Stopped capture\n");
    print_objc_averages();
}