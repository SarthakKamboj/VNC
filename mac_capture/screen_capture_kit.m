#include "screen_capture_kit.h"

#import <ScreenCaptureKit/ScreenCaptureKit.h>

void* create_sc_stream_configuration() {
    SCStreamConfiguration *config = [[SCStreamConfiguration alloc] init];
    return (__bridge_retained void*)config;
}