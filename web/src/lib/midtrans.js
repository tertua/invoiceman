let snapPromise;

export function loadMidtransSnap(isProduction = false) {
  if (window.snap) return Promise.resolve(window.snap);
  if (snapPromise) return snapPromise;
  snapPromise = new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = `https://${isProduction ? "app" : "app.sandbox"}.midtrans.com/snap/snap.js`;
    script.async = true;
    script.onload = () => window.snap ? resolve(window.snap) : reject(new Error("Midtrans Snap failed to load"));
    script.onerror = () => reject(new Error("Midtrans Snap failed to load"));
    document.head.appendChild(script);
  });
  return snapPromise;
}
