/**
 * Makes a phone photo small before upload: at most `max` pixels on the long side, JPEG.
 * The browser applies the photo's EXIF rotation while decoding, and the new file has no
 * EXIF data at all (so no GPS location leaks).
 */
export async function shrinkPhoto(file: File, max = 1200): Promise<Blob> {
  try {
    const bitmap = await createImageBitmap(file, { imageOrientation: "from-image" });
    const scale = Math.min(1, max / Math.max(bitmap.width, bitmap.height));
    const canvas = document.createElement("canvas");
    canvas.width = Math.round(bitmap.width * scale);
    canvas.height = Math.round(bitmap.height * scale);
    canvas.getContext("2d")!.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
    bitmap.close();
    return await new Promise((resolve, reject) =>
      canvas.toBlob((blob) => (blob ? resolve(blob) : reject(new Error("encode failed"))), "image/jpeg", 0.85),
    );
  } catch {
    // Formats the browser cannot draw (rare): send the original; the server checks it.
    return file;
  }
}
