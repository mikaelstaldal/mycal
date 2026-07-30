// Mirrors internal/repository — the demo's storage layer.
//
// Where the server has SQLite, the demo has one IndexedDB value: the whole
// emulated database (events, calendars, preferences and the id counters) as a
// single DemoState document under the key "state". A demo holds a few dozen
// rows, so there is nothing to gain from splitting it into object stores, and
// keeping it in one value makes a multi-row change — deleting a recurring
// parent along with its overrides, say — atomic without a cross-store
// transaction.
//
// All access goes through withStore, which serialises operations on a promise
// chain. A service worker handles fetch events concurrently, so two overlapping
// writes would otherwise read the same state, modify their own copy, and have
// the second write discard the first.

const DB_NAME = 'mycal-demo';
const DB_VERSION = 1;
const OBJECT_STORE = 'kv';
const STATE_KEY = 'state';

/**
 * The DemoState format this build writes. A future change that cannot be read
 * by an older worker bumps this; loadState discards a document from the future
 * rather than misreading it.
 */
const STATE_VERSION = 1;

let dbPromise: Promise<IDBDatabase> | null = null;
/** The loaded state, kept in memory so a read does not hit IndexedDB every time. */
let cachedState: DemoState | null = null;
/** The tail of the operation chain that serialises access (see withStore). */
let queue: Promise<unknown> = Promise.resolve();

/** Opens the database, creating the object store on first use. */
function openDatabase(): Promise<IDBDatabase> {
    if (dbPromise !== null) return dbPromise;
    dbPromise = new Promise<IDBDatabase>((resolve, reject) => {
        const request = indexedDB.open(DB_NAME, DB_VERSION);
        request.onupgradeneeded = () => {
            const db = request.result;
            if (!db.objectStoreNames.contains(OBJECT_STORE)) db.createObjectStore(OBJECT_STORE);
        };
        request.onsuccess = () => resolve(request.result);
        request.onerror = () => reject(request.error ?? new Error('cannot open the demo database'));
        request.onblocked = () => reject(new Error('the demo database is blocked by another tab'));
    }).catch((err: unknown) => {
        dbPromise = null; // let a later request try again
        throw storageError(err);
    });
    return dbPromise;
}

/** Reads one value from the key-value store. */
function idbGet<T>(key: string): Promise<T | undefined> {
    return openDatabase().then(
        (db) =>
            new Promise<T | undefined>((resolve, reject) => {
                const tx = db.transaction(OBJECT_STORE, 'readonly');
                const request = tx.objectStore(OBJECT_STORE).get(key);
                request.onsuccess = () => resolve(request.result as T | undefined);
                request.onerror = () => reject(storageError(request.error));
            }),
    );
}

/** Writes one value to the key-value store. */
function idbPut(key: string, value: unknown): Promise<void> {
    return openDatabase().then(
        (db) =>
            new Promise<void>((resolve, reject) => {
                const tx = db.transaction(OBJECT_STORE, 'readwrite');
                tx.objectStore(OBJECT_STORE).put(value, key);
                tx.oncomplete = () => resolve();
                tx.onabort = () => reject(storageError(tx.error));
                tx.onerror = () => reject(storageError(tx.error));
            }),
    );
}

/**
 * Turns a storage failure into something the UI can show. A full quota is the
 * one failure a visitor can actually cause, and losing a write silently is the
 * one outcome worth avoiding, so it gets its own status.
 */
function storageError(err: unknown): ApiError {
    if (err instanceof ApiError) return err;
    const name = err instanceof Error ? err.name : '';
    if (name === 'QuotaExceededError') {
        return new ApiError(507, 'demo storage is full: free up browser storage for this site and try again');
    }
    const message = err instanceof Error ? err.message : String(err);
    return new ApiError(500, 'demo storage error: ' + message);
}

/** A fresh database: the reserved default calendar and nothing else (schemaV1). */
function initialState(): DemoState {
    return {
        version: STATE_VERSION,
        next_event_id: 1,
        next_calendar_id: 1,
        events: [],
        calendars: [{ ...DEFAULT_CALENDAR }],
        preferences: {},
    };
}

/** Loads the stored state, creating and persisting an empty one on first run. */
async function loadState(): Promise<DemoState> {
    if (cachedState !== null) return cachedState;
    const stored = await idbGet<DemoState>(STATE_KEY);
    if (stored !== undefined && stored.version === STATE_VERSION) {
        cachedState = stored;
        return stored;
    }
    const state = initialState();
    cachedState = state;
    await idbPut(STATE_KEY, state);
    return state;
}

/**
 * Runs fn against the stored state with no other operation interleaved. A write
 * persists the state fn leaves behind; a read does not, so a handler that only
 * lists events cannot accidentally write.
 *
 * Every handler in api.ts goes through here, which is what makes a
 * read-modify-write — appending an EXDATE, allocating an id — safe against a
 * second request arriving mid-flight.
 */
function withStore<T>(mode: 'read' | 'write', fn: (state: DemoState) => T | Promise<T>): Promise<T> {
    const run = queue.then(async () => {
        const state = await loadState();
        const result = await fn(state);
        if (mode === 'write') {
            try {
                await idbPut(STATE_KEY, state);
            } catch (err) {
                throw storageError(err);
            }
        }
        return result;
    });
    // The chain must survive a failed operation, so swallow the result here;
    // the caller still sees the rejection through `run`.
    queue = run.then(
        () => undefined,
        () => undefined,
    );
    return run;
}

/** Allocates the next event id, standing in for SQLite's AUTOINCREMENT. */
function nextEventID(state: DemoState): number {
    const id = state.next_event_id;
    state.next_event_id = id + 1;
    return id;
}

/** Allocates the next calendar id. */
function nextCalendarID(state: DemoState): number {
    const id = state.next_calendar_id;
    state.next_calendar_id = id + 1;
    return id;
}

/** The timestamp SQLite would write into created_at / updated_at. */
function nowRFC3339(): string {
    return formatRFC3339(new Date());
}

/**
 * Makes sure the store exists, and is where the seeded demo content will be
 * written on first run. The worker calls this on activate so the first request
 * finds a populated database rather than paying for the seed itself.
 */
async function seedStore(): Promise<void> {
    await withStore('read', () => undefined);
}
