const accepted = ["image/jpeg", "image/png", "image/webp"]

export function isHeic(file: File): boolean {
  return /image\/hei[cf]/i.test(file.type) || /\.hei[cf]$/i.test(file.name)
}

// FR-MEDIA-01 and FR-MEDIA-03: convert iPhone photos and shrink before upload, heavy libraries load only when needed
export async function prepareImage(file: File): Promise<File> {
  let input: File = file
  if (isHeic(file)) {
    const { heicTo } = await import("heic-to")
    const blob = await heicTo({ blob: file, type: "image/jpeg", quality: 0.9 })
    input = new File([blob], file.name.replace(/\.hei[cf]$/i, ".jpg"), { type: "image/jpeg" })
  }
  if (!accepted.includes(input.type)) {
    throw new Error("unsupported")
  }
  const { default: compress } = await import("browser-image-compression")
  // the worker mode fetches its script from a public CDN, the main thread keeps every byte on our origin
  return compress(input, { maxWidthOrHeight: 2048, initialQuality: 0.85, useWebWorker: false, maxSizeMB: 9 })
}
