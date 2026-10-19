# Upload validation

Call `Validate(w, r, Limits{MinBytes: 1, MaxBytes: 26214400, MaxPixels: 20000000})`
with a raw file request body. For multipart uploads, the HTTP layer must separately
limit the entire request and extract the file part before using this validator.
Limits are inclusive. The validator replaces and closes the request body.

The body streams through `http.MaxBytesReader` into a private temporary file.
Neither Content-Length nor the client Content-Type determines validity. Bytes
after the image header still count toward the size limit. The first 512 bytes
identify JPEG, PNG, WebP, or HEIC; filenames and extensions are ignored.

On success the returned File exposes MIME, Size, Width, and Height and is rewound
for streaming into incoming storage. Close it after storage finishes to remove
the temporary file. Failures also remove the file. TempDir can select a disk-backed
volume; otherwise the operating system's temporary directory is used.

Dimensions come from `image.DecodeConfig`, with an overflow-safe pixel comparison.
JPEG and PNG use Go's standard decoders. WebP and HEIC register configuration-only
readers; they do not implement `image.Decode` pixel decoding. No native library
or additional dependency is required by this package.

WebP supports VP8, VP8L, and VP8X headers according to
[RFC 9649](https://www.rfc-editor.org/rfc/rfc9649.html).
HEIC requires a HEVC-specific file brand within the sniffed bytes and reads
`meta/iprp/ipco/ispe` extents, as represented by
[libheif's box implementation](https://github.com/strukturag/libheif/blob/master/libheif/box.cc).
For multi-item HEIC files Width and Height conservatively bound all declared
extents, including auxiliary images, rather than selecting a primary item.
Container nesting, box count, and box lengths are checked without allocating
buffers based on untrusted dimensions or metadata lengths.

These are header and declared-pixel-count checks, not full compressed-bitstream
validation. Later image processing must also enforce decoder allocation limits;
HEVC bitstream dimensions can disagree with container metadata. Files stay in
private incoming storage until scanning and processing pass.

Run `go test ./internal/validate`. WebP/HEIC fixtures test metadata parsing;
JPEG/PNG fixtures are encoded images, including a PNG with oversized dimensions.
