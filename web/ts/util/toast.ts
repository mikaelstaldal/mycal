export interface ToastItem {
  id: number;
  message: string;
  persistent: boolean;
  error: boolean;
  onRetry?: () => void;
}

type Listener = (items: ToastItem[]) => void;

let items: ToastItem[] = [];
let nextId = 0;
const listeners = new Set<Listener>();

function notify(): void {
  const snapshot = [...items];
  for (const l of listeners) l(snapshot);
}

export function subscribe(listener: Listener): () => void {
  listeners.add(listener);
  listener([...items]);
  return () => { listeners.delete(listener); };
}

export function showToast(message: string, opts: { error?: boolean; duration?: number } = {}): void {
  const id = nextId++;
  const duration = opts.duration ?? (opts.error ? 5000 : 3000);
  items = [...items, { id, message, persistent: false, error: !!opts.error }];
  notify();
  setTimeout(() => dismissToast(id), duration);
}

export function showNetworkErrorToast(message: string, onRetry: () => void): void {
  const id = nextId++;
  items = [...items, { id, message, persistent: true, error: true, onRetry }];
  notify();
}

export function dismissToast(id: number): void {
  items = items.filter(t => t.id !== id);
  notify();
}
