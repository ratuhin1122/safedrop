//go:build cgo && !windows

#include "native.h"

int sd_probe(const char *input, int *width, int *height) {
    VipsImage *image = vips_image_new_from_file(input,
        "access", VIPS_ACCESS_SEQUENTIAL, "fail_on", VIPS_FAIL_ON_ERROR, NULL);
    if (!image) return -1;
    *width = vips_image_get_width(image);
    *height = vips_image_get_height(image);
    int orientation = 1;
    if (vips_image_get_typeof(image, VIPS_META_ORIENTATION))
        vips_image_get_int(image, VIPS_META_ORIENTATION, &orientation);
    if (orientation >= 5 && orientation <= 8) {
        int value = *width; *width = *height; *height = value;
    }
    g_object_unref(image);
    return 0;
}

int sd_encode(const char *input, const char *output, int width, int height, int avif) {
    VipsImage *source = vips_image_new_from_file(input,
        "access", VIPS_ACCESS_SEQUENTIAL, "fail_on", VIPS_FAIL_ON_ERROR, NULL);
    VipsImage *rotated = NULL, *resized = NULL;
    int result = -1;
    if (!source) return -1;
    if (vips_autorot(source, &rotated, NULL)) goto done;
    if (vips_resize(rotated, &resized,
        (double) width / vips_image_get_width(rotated),
        "vscale", (double) height / vips_image_get_height(rotated),
        "kernel", VIPS_KERNEL_LANCZOS3, NULL)) goto done;
    if (avif)
        result = vips_heifsave(resized, output, "Q", 80, "effort", 4,
            "compression", VIPS_FOREIGN_HEIF_COMPRESSION_AV1, "strip", TRUE, NULL);
    else
        result = vips_webpsave(resized, output, "Q", 80, "effort", 4,
            "strip", TRUE, NULL);
done:
    if (resized) g_object_unref(resized);
    if (rotated) g_object_unref(rotated);
    g_object_unref(source);
    return result;
}
