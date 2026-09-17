#ifndef SCREEN_CAPTURE_KIT
#define SCREEN_CAPTURE_KIT

typedef struct pixel_t {
    float r;
    float g;
    float b;
} pixel_t;

typedef struct frame_t {
    // pixel_t* pixel_data;    
    int width;
    int height;
    int num_planes;
} frame_t;

void start_capture();

#endif // SCREEN_CAPTURE_KIT