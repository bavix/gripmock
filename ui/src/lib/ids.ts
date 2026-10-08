export function randomHex(chars = 8): string {
  const bytes = new Uint8Array(Math.ceil(chars / 2));
  const webCrypto = globalThis.crypto;

  if (typeof webCrypto?.getRandomValues === 'function') {
    webCrypto.getRandomValues(bytes);
  } else {
    for (let i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256);
  }

  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('').slice(0, chars);
}

export function newSessionId(): string {
  return `sess-${randomHex(8)}`;
}
