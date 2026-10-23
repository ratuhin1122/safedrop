export class MockXHR {
  static requests: MockXHR[] = [];
  upload: { onprogress: ((event: { loaded: number; total: number; lengthComputable: boolean }) => void) | null } = { onprogress: null };
  onload: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onabort: (() => void) | null = null;
  ontimeout: (() => void) | null = null;
  headers = new Headers();
  status = 0;
  responseText = "";
  timeout = 0;
  method = "";
  url = "";
  body: FormData | null = null;
  aborted = false;
  retryAfter: string | null = null;
  open(method: string, url: string) { this.method = method; this.url = url; }
  setRequestHeader(key: string, value: string) { this.headers.set(key, value); }
  getResponseHeader(name: string) { return name === "Retry-After" ? this.retryAfter : null; }
  send(body: FormData) { this.body = body; MockXHR.requests.push(this); }
  abort() { this.aborted = true; this.onabort?.(); }
  progress(loaded: number, total = 100) { this.upload.onprogress?.({ loaded, total, lengthComputable: true }); }
  respond(status: number, body: unknown, retryAfter: string | null = null) {
    this.status = status;
    this.responseText = JSON.stringify(body);
    this.retryAfter = retryAfter;
    this.onload?.();
  }
}
