#ifndef SAFEDROP_IMAGE_NATIVE_H
#define SAFEDROP_IMAGE_NATIVE_H
#include <vips/vips.h>
int sd_probe(const char *input, int *width, int *height);
int sd_encode(const char *input, const char *output, int width, int height, int avif);
#endif
