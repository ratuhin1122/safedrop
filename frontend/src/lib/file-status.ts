export const fileSteps = ["uploaded", "scanning", "processing", "ready"] as const;
export const stepLabels = { uploaded: "Uploaded", scanning: "Scanning", processing: "Processing", ready: "Ready" };

export function rejectionMessage(reason: string | null) {
  switch (reason) {
    case "virus": case "rejected_virus": return "A virus was found. This file cannot be shared.";
    case "invalid_image": case "bad_type": case "rejected_type": return "This image could not be read. Try a valid JPEG, PNG, WebP or HEIC image.";
    case "pixel_limit": case "too_many_pixels": case "rejected_pixels": return "This image's dimensions are too large. Try a smaller image.";
    case "processing_error": return "We couldn't process this file. Try a different image.";
    default: return "This file was rejected and cannot be shared. Try a different image.";
  }
}
