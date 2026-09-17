#ifndef SCREEN_CAPTURE_KIT
#define SCREEN_CAPTURE_KIT

#include <stdint.h>

typedef struct pixel_t {
    uint8_t r;
    uint8_t g;
    uint8_t b;
    uint8_t a;
} pixel_t;

typedef struct frame_t {
    pixel_t* pixel_data;    
    int width;
    int height;
    int num_planes;
} frame_t;

void start_capture();

#endif // SCREEN_CAPTURE_KIT