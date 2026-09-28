// Save an image to the payer's device (QR download on pay widgets).
// data: URLs save directly; remote URLs go through fetch→blob so the file
// actually downloads instead of navigating away (plain `download` attr is
// ignored cross-origin). Returns true on success, false when the caller
// should fall back (e.g. open the image in a new tab).
export async function downloadImage(src, filename) {
  if (!src) return false;
  try {
    if (src.startsWith("data:")) {
      const a = document.createElement("a");
      a.href = src;
      a.download = filename || "qr.png";
      document.body.appendChild(a);
      a.click();
      a.remove();
      return true;
    }
    const res = await fetch(src, { mode: "cors" });
    if (!res.ok) return false;
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename || "qr.png";
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(url), 4000);
    return true;
  } catch {
    return false;
  }
}
