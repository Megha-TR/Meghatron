export async function fetchJSON<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, { ...options, signal: options.signal ?? AbortSignal.timeout(10000) });
  if (!response.ok) {
    let message = `API request failed (${response.status})`;
    try { const body = await response.json(); if (typeof body.error === 'string') message = body.error; } catch { /* A proxy can return a non-JSON error. */ }
    throw new Error(message);
  }
  return response.json() as Promise<T>;
}
